// Package git provides git operations for DevFlow managed projects.
// All operations shell out to the system git binary.
package git

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AuthConfig holds authentication for a single git operation.
type AuthConfig struct {
	// SSH key auth: provide PrivateKeyPEM (PEM-encoded private key bytes).
	PrivateKeyPEM []byte
	// Token auth: provide Token (PAT or deploy token) for HTTPS repos.
	Token string
}

// CloneOptions controls git clone behavior.
type CloneOptions struct {
	RepoURL       string
	TargetDir     string
	Branch        string     // empty = default branch
	Auth          AuthConfig
	Depth         int        // 0 = full history; >0 = shallow clone
	Timeout       time.Duration
}

// PullOptions controls git pull behavior.
type PullOptions struct {
	RepoDir string
	Branch  string
	Auth    AuthConfig
	Timeout time.Duration
}

// BranchOptions controls branch creation.
type BranchOptions struct {
	RepoDir    string
	BranchName string
	Base       string // branch/commit to base from; empty = HEAD
}

// Clone clones a repository into TargetDir.
// If TargetDir already exists and is a git repo, returns nil (idempotent).
func Clone(ctx context.Context, opts CloneOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// If target already exists as a git repo, skip clone.
	if isGitRepo(opts.TargetDir) {
		slog.Info("devflow.git.clone_skipped", "dir", opts.TargetDir, "reason", "already a git repo")
		return nil
	}

	repoURL, cleanup, err := prepareURL(opts.RepoURL, opts.Auth)
	if err != nil {
		return fmt.Errorf("prepare repo url: %w", err)
	}
	defer cleanup()

	args := []string{"clone", "--progress"}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	args = append(args, repoURL, opts.TargetDir)

	env, sshCleanup, err := buildEnv(opts.Auth)
	if err != nil {
		return fmt.Errorf("setup ssh: %w", err)
	}
	defer sshCleanup()

	out, err := run(ctx, "", env, "git", args...)
	if err != nil {
		return fmt.Errorf("git clone: %w\noutput: %s", err, out)
	}
	slog.Info("devflow.git.cloned", "dir", opts.TargetDir, "url", sanitizeURL(opts.RepoURL))
	return nil
}

// Pull fetches + merges latest changes in an existing repo.
func Pull(ctx context.Context, opts PullOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if !isGitRepo(opts.RepoDir) {
		return fmt.Errorf("not a git repository: %s", opts.RepoDir)
	}

	env, sshCleanup, err := buildEnv(opts.Auth)
	if err != nil {
		return fmt.Errorf("setup ssh: %w", err)
	}
	defer sshCleanup()

	// First fetch
	if out, err := run(ctx, opts.RepoDir, env, "git", "fetch", "--all", "--prune"); err != nil {
		return fmt.Errorf("git fetch: %w\noutput: %s", err, out)
	}

	// Then merge (fast-forward only)
	pullArgs := []string{"pull", "--ff-only"}
	if opts.Branch != "" {
		pullArgs = append(pullArgs, "origin", opts.Branch)
	}
	if out, err := run(ctx, opts.RepoDir, env, "git", pullArgs...); err != nil {
		return fmt.Errorf("git pull: %w\noutput: %s", err, out)
	}

	slog.Info("devflow.git.pulled", "dir", opts.RepoDir)
	return nil
}

// CreateBranch creates and checks out a new branch.
// If the branch already exists, switches to it.
func CreateBranch(ctx context.Context, opts BranchOptions) error {
	if !isGitRepo(opts.RepoDir) {
		return fmt.Errorf("not a git repository: %s", opts.RepoDir)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Check if branch already exists
	out, _ := run(ctx, opts.RepoDir, nil, "git", "branch", "--list", opts.BranchName)
	if strings.TrimSpace(string(out)) != "" {
		// Branch exists — just checkout
		if out, err := run(ctx, opts.RepoDir, nil, "git", "checkout", opts.BranchName); err != nil {
			return fmt.Errorf("git checkout: %w\noutput: %s", err, out)
		}
		return nil
	}

	// Create new branch
	args := []string{"checkout", "-b", opts.BranchName}
	if opts.Base != "" {
		args = append(args, opts.Base)
	}
	if out, err := run(ctx, opts.RepoDir, nil, "git", args...); err != nil {
		return fmt.Errorf("git checkout -b: %w\noutput: %s", err, out)
	}

	slog.Info("devflow.git.branch_created", "dir", opts.RepoDir, "branch", opts.BranchName)
	return nil
}

// CurrentBranch returns the current branch name in a repo.
func CurrentBranch(ctx context.Context, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := run(ctx, repoDir, nil, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------

func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

// buildEnv constructs the environment for a git command.
// Always disables terminal prompts so failures are immediate and clear.
// For SSH key auth, writes key to a temp file and sets GIT_SSH_COMMAND.
// Returns cleanup function that must be called (even on error).
func buildEnv(auth AuthConfig) ([]string, func(), error) {
	base := append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", // never prompt for credentials
		"GIT_ASKPASS=true",      // return empty string on credential ask → fast fail
	)

	if len(auth.PrivateKeyPEM) == 0 {
		return base, func() {}, nil
	}

	// Write private key to a temp file with strict permissions
	f, err := os.CreateTemp("", "devflow-ssh-*.pem")
	if err != nil {
		return nil, func() {}, fmt.Errorf("create ssh key temp file: %w", err)
	}
	keyPath := f.Name()

	if err := os.Chmod(keyPath, 0600); err != nil {
		f.Close()
		os.Remove(keyPath)
		return nil, func() {}, fmt.Errorf("chmod ssh key: %w", err)
	}
	// Normalize key: strip \r, ensure trailing newline — OpenSSH is strict about format
	keyData := strings.ReplaceAll(string(auth.PrivateKeyPEM), "\r\n", "\n")
	keyData = strings.ReplaceAll(keyData, "\r", "\n")
	keyData = strings.TrimSpace(keyData) + "\n"
	if _, err := f.WriteString(keyData); err != nil {
		f.Close()
		os.Remove(keyPath)
		return nil, func() {}, fmt.Errorf("write ssh key: %w", err)
	}
	f.Close()

	cleanup := func() { os.Remove(keyPath) }

	sshCmd := fmt.Sprintf(
		`ssh -i %s -o StrictHostKeyChecking=no -o BatchMode=yes -o IdentitiesOnly=yes`,
		keyPath,
	)
	env := append(base, "GIT_SSH_COMMAND="+sshCmd)
	return env, cleanup, nil
}

// prepareURL returns the repo URL with token embedded for HTTPS, or as-is for SSH.
// cleanup must always be called (noop for non-token auth).
//
// Username mapping:
//   - bitbucket.org → x-token-auth  (HTTP access tokens)
//   - all others    → oauth2         (GitHub, GitLab, Gitea, etc.)
func prepareURL(repoURL string, auth AuthConfig) (string, func(), error) {
	noop := func() {}
	if auth.Token == "" {
		return repoURL, noop, nil
	}
	// Only embed token for HTTPS URLs
	if !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "http://") {
		return repoURL, noop, nil
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", noop, fmt.Errorf("parse repo url: %w", err)
	}
	username := "oauth2"
	if strings.Contains(u.Host, "bitbucket.org") {
		username = "x-token-auth"
	}
	u.User = url.UserPassword(username, auth.Token)
	return u.String(), noop, nil
}

// sanitizeURL removes credentials from a URL for logging.
// Handles both HTTPS (https://user:pass@host/path) and SSH (git@host:org/repo) formats.
func sanitizeURL(rawURL string) string {
	// SSH-style URLs (git@host:org/repo) don't parse with url.Parse
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		// e.g. git@bitbucket.org:team/repo.git → bitbucket.org:team/repo.git
		if at := strings.Index(rawURL, "@"); at >= 0 {
			return rawURL[at+1:]
		}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = nil
	return u.String()
}

// run executes a git command, returning combined output.
func run(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.Bytes(), err
}
