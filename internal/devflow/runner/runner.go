// Package runner spawns Claude Code CLI processes for DevFlow runs.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PermissionMode maps to claude --permission-mode values.
type PermissionMode string

const (
	PermissionModeDefault            PermissionMode = "default"
	PermissionModeAcceptEdits        PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions  PermissionMode = "bypassPermissions"
	PermissionModeDontAsk            PermissionMode = "dontAsk"
)

// RunOptions configures a single Claude Code run.
type RunOptions struct {
	// WorkspaceDir is the project workspace (repo clone path).
	WorkspaceDir string
	// SessionID is a pre-generated UUID used for tracking in ext_devflow_runs.
	// Also passed to claude via --session-id so we can correlate logs.
	SessionID string
	// TaskPrompt is the user-facing task description passed as the claude prompt.
	TaskPrompt string
	// ContextPrompt is tier-2 context injected via --append-system-prompt.
	// Typically contains relevant doc excerpts for this specific task.
	ContextPrompt string
	// Branch: if non-empty, uses --worktree <branch> to isolate edits in a
	// separate git worktree. Leave empty to work directly in WorkspaceDir.
	Branch string
	// PermissionMode controls tool permission prompting.
	// Use BypassPermissions for fully automated runs.
	PermissionMode PermissionMode
	// AgentTeams enables CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1.
	AgentTeams bool
	// MaxBudgetUSD caps spend per run (0 = no limit).
	MaxBudgetUSD float64
	// Model overrides the default claude model.
	Model string
	// ClaudeBin is the path to the claude binary. Defaults to "claude".
	ClaudeBin string
	// Timeout for the entire run. Defaults to 30 minutes.
	Timeout time.Duration
	// ExtraEnv is merged into the subprocess environment.
	ExtraEnv []string
	// Stdout receives streamed output (optional; captured internally if nil).
	Stdout io.Writer
}

// Result is returned from Execute.
type Result struct {
	// Output is the final text response from Claude.
	Output string
	// SessionID echoed back (may differ from opts.SessionID if claude allocated one).
	SessionID string
	// TotalCostUSD from the JSON output (0 if not available).
	TotalCostUSD float64
	// ExitCode of the claude process.
	ExitCode int
}

// claudeJSONOutput is the structured output from claude --output-format json.
type claudeJSONOutput struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	CostUSD   float64 `json:"cost_usd"`
}

// Execute spawns a claude CLI process and waits for completion.
// It is safe to call from multiple goroutines (each call is independent).
func Execute(ctx context.Context, opts RunOptions) (*Result, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	claudeBin := opts.ClaudeBin
	if claudeBin == "" {
		claudeBin = "claude"
	}

	if opts.SessionID == "" {
		opts.SessionID = uuid.New().String()
	}

	args := buildArgs(opts)

	cmd := exec.CommandContext(ctx, claudeBin, args...)
	cmd.Dir = opts.WorkspaceDir
	cmd.Env = buildEnv(opts)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	if opts.Stdout != nil {
		cmd.Stdout = io.MultiWriter(&outBuf, opts.Stdout)
	} else {
		cmd.Stdout = &outBuf
	}
	cmd.Stderr = &errBuf

	slog.Info("devflow.runner.start",
		"session_id", opts.SessionID,
		"workspace", opts.WorkspaceDir,
		"branch", opts.Branch,
		"agent_teams", opts.AgentTeams,
		"task_preview", truncate(opts.TaskPrompt, 120),
	)

	startedAt := time.Now()
	err := cmd.Run()
	elapsed := time.Since(startedAt)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	rawOut := outBuf.String()
	rawErr := errBuf.String()

	slog.Info("devflow.runner.done",
		"session_id", opts.SessionID,
		"exit_code", exitCode,
		"elapsed_s", int(elapsed.Seconds()),
	)

	if exitCode != 0 && rawErr != "" {
		slog.Warn("devflow.runner.stderr", "session_id", opts.SessionID, "stderr", truncate(rawErr, 500))
	}

	result := &Result{
		ExitCode:  exitCode,
		SessionID: opts.SessionID,
	}

	// Try to parse JSON output; fall back to raw text.
	if parsed, ok := parseJSONOutput(rawOut); ok {
		result.Output = parsed.Result
		result.TotalCostUSD = parsed.CostUSD
		if parsed.SessionID != "" {
			result.SessionID = parsed.SessionID
		}
	} else {
		result.Output = rawOut
	}

	if err != nil && exitCode != 0 {
		return result, fmt.Errorf("claude exited %d: %s", exitCode, truncate(rawErr, 300))
	}

	return result, nil
}

// buildArgs constructs the claude CLI argument list from RunOptions.
func buildArgs(opts RunOptions) []string {
	args := []string{"--print", "--output-format", "json"}

	// Pre-set session ID for correlation
	if opts.SessionID != "" {
		args = append(args, "--session-id", opts.SessionID)
	}

	// Tier-2 context injection
	if opts.ContextPrompt != "" {
		args = append(args, "--append-system-prompt", opts.ContextPrompt)
	}

	// Git worktree isolation per task
	if opts.Branch != "" {
		args = append(args, "--worktree", opts.Branch)
	}

	// Permission mode
	mode := opts.PermissionMode
	if mode == "" {
		mode = PermissionModeBypassPermissions
	}
	args = append(args, "--permission-mode", string(mode))

	// Budget cap
	if opts.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.MaxBudgetUSD))
	}

	// Model override
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Task prompt (positional argument, must be last)
	args = append(args, opts.TaskPrompt)

	return args
}

// buildEnv constructs the subprocess environment.
func buildEnv(opts RunOptions) []string {
	env := os.Environ()

	if opts.AgentTeams {
		env = append(env, "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1")
	}

	// Disable interactive prompts in CI/automated mode
	env = append(env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")

	env = append(env, opts.ExtraEnv...)
	return env
}

// parseJSONOutput tries to extract the structured result from claude --output-format json.
// Claude prints one JSON object per line; we look for the "result" type.
func parseJSONOutput(raw string) (claudeJSONOutput, bool) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	// Scan from last line backwards — the final result object comes last
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var out claudeJSONOutput
		if err := json.Unmarshal([]byte(line), &out); err == nil {
			if out.Result != "" || out.Type == "result" {
				return out, true
			}
		}
	}
	return claudeJSONOutput{}, false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
