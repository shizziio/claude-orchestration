package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	devgit "github.com/nextlevelbuilder/goclaw/internal/devflow/git"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowGitOpsHandler handles git clone/pull/branch operations for DevFlow projects.
type DevflowGitOpsHandler struct {
	projects      store.ProjectStore
	gitCreds      store.GitCredentialStore // also used for key/token decryption
	workspaceBase string
}

func NewDevflowGitOpsHandler(
	projects store.ProjectStore,
	gitCreds store.GitCredentialStore,
	workspaceBase string,
) *DevflowGitOpsHandler {
	return &DevflowGitOpsHandler{
		projects:      projects,
		gitCreds:      gitCreds,
		workspaceBase: workspaceBase,
	}
}

func (h *DevflowGitOpsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/devflow/projects/{id}/clone", requireAuth("", h.handleClone))
	mux.HandleFunc("POST /v1/devflow/projects/{id}/pull", requireAuth("", h.handlePull))
	mux.HandleFunc("POST /v1/devflow/projects/{id}/branch", requireAuth("", h.handleBranch))
	mux.HandleFunc("GET /v1/devflow/projects/{id}/branch", requireAuth("", h.handleCurrentBranch))
}

func (h *DevflowGitOpsHandler) handleClone(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if p.RepoURL == nil || *p.RepoURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project has no repo_url"})
		return
	}

	var body struct {
		Branch string `json:"branch"`
		Depth  int    `json:"depth"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	workspaceDir := h.resolveWorkspace(p)

	if err := os.MkdirAll(filepath.Dir(workspaceDir), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create workspace dir: %s", err)})
		return
	}

	auth, err := h.resolveAuth(r.Context(), p)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("resolve auth: %s", err)})
		return
	}

	branch := body.Branch
	if branch == "" {
		branch = p.DefaultBranch
	}

	if err := devgit.Clone(r.Context(), devgit.CloneOptions{
		RepoURL:   *p.RepoURL,
		TargetDir: workspaceDir,
		Branch:    branch,
		Auth:      auth,
		Depth:     body.Depth,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Persist workspace path back to project
	_, _ = h.projects.Update(r.Context(), id, store.UpdateProjectInput{
		WorkspacePath: &workspaceDir,
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"status":         "cloned",
		"workspace_path": workspaceDir,
	})
}

func (h *DevflowGitOpsHandler) handlePull(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if p.WorkspacePath == nil || *p.WorkspacePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project not cloned yet — run clone first"})
		return
	}

	var body struct {
		Branch string `json:"branch"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	auth, err := h.resolveAuth(r.Context(), p)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("resolve auth: %s", err)})
		return
	}

	branch := body.Branch
	if branch == "" {
		branch = p.DefaultBranch
	}

	if err := devgit.Pull(r.Context(), devgit.PullOptions{
		RepoDir: *p.WorkspacePath,
		Branch:  branch,
		Auth:    auth,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "pulled"})
}

func (h *DevflowGitOpsHandler) handleBranch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if p.WorkspacePath == nil || *p.WorkspacePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project not cloned yet"})
		return
	}
	var body struct {
		Branch string `json:"branch"`
		Base   string `json:"base"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Branch == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "branch is required"})
		return
	}
	if err := devgit.CreateBranch(r.Context(), devgit.BranchOptions{
		RepoDir:    *p.WorkspacePath,
		BranchName: body.Branch,
		Base:       body.Base,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "branch": body.Branch})
}

func (h *DevflowGitOpsHandler) handleCurrentBranch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if p.WorkspacePath == nil || *p.WorkspacePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project not cloned yet"})
		return
	}
	branch, err := devgit.CurrentBranch(r.Context(), *p.WorkspacePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"branch": branch})
}

// resolveWorkspace returns the local workspace directory for a project.
func (h *DevflowGitOpsHandler) resolveWorkspace(p *store.Project) string {
	if p.WorkspacePath != nil && *p.WorkspacePath != "" {
		return *p.WorkspacePath
	}
	base := h.workspaceBase
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, "devflow", "workspaces")
	}
	return filepath.Join(base, p.Slug)
}

// resolveAuth builds git AuthConfig from the project's linked credential.
func (h *DevflowGitOpsHandler) resolveAuth(ctx context.Context, p *store.Project) (devgit.AuthConfig, error) {
	if p.GitCredentialID == nil {
		return devgit.AuthConfig{}, nil
	}

	cred, err := h.gitCreds.Get(ctx, *p.GitCredentialID)
	if err != nil {
		return devgit.AuthConfig{}, fmt.Errorf("get credential: %w", err)
	}

	switch cred.AuthType {
	case "ssh_key":
		key, err := h.gitCreds.GetPrivateKey(ctx, cred.ID)
		if err != nil {
			return devgit.AuthConfig{}, fmt.Errorf("read private key: %w", err)
		}
		return devgit.AuthConfig{PrivateKeyPEM: key}, nil
	case "token":
		token, err := h.gitCreds.GetToken(ctx, cred.ID)
		if err != nil {
			return devgit.AuthConfig{}, fmt.Errorf("read token: %w", err)
		}
		return devgit.AuthConfig{Token: string(token)}, nil
	default:
		return devgit.AuthConfig{}, fmt.Errorf("unknown auth_type: %s", cred.AuthType)
	}
}
