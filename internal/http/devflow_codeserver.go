package http

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/devflow/codeserver"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

const codeServerBasePort = 18800

// DevflowCodeServerHandler manages code-server start/stop/status for environments.
type DevflowCodeServerHandler struct {
	envs     store.EnvironmentStore
	projects store.ProjectStore
	mgr      *codeserver.Manager
}

func NewDevflowCodeServerHandler(envs store.EnvironmentStore, projects store.ProjectStore, mgr *codeserver.Manager) *DevflowCodeServerHandler {
	return &DevflowCodeServerHandler{envs: envs, projects: projects, mgr: mgr}
}

func (h *DevflowCodeServerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/environments/{envId}/code-server/start", requireAuth("", h.handleStart))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/environments/{envId}/code-server/stop", requireAuth("", h.handleStop))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/environments/{envId}/code-server/status", requireAuth("", h.handleStatus))
}

func (h *DevflowCodeServerHandler) resolveEnv(r *http.Request) (*store.Project, *store.Environment, error) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid projectId")
	}
	envID, err := uuid.Parse(r.PathValue("envId"))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid envId")
	}
	project, err := h.projects.Get(r.Context(), projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("project not found")
	}
	env, err := h.envs.Get(r.Context(), envID)
	if err != nil {
		return nil, nil, fmt.Errorf("environment not found")
	}
	if project.WorkspacePath == nil || *project.WorkspacePath == "" {
		return nil, nil, fmt.Errorf("project workspace not set — clone the repo first")
	}
	return project, env, nil
}

func (h *DevflowCodeServerHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	project, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// If already has a port and is running, return it
	if env.CodeServerPort != nil && *env.CodeServerPort > 0 && h.mgr.IsRunning(*env.CodeServerPort) {
		writeJSON(w, http.StatusOK, map[string]any{
			"running": true,
			"port":    *env.CodeServerPort,
			"url":     fmt.Sprintf("http://localhost:%d", *env.CodeServerPort),
		})
		return
	}

	// Pick a port and start
	port := h.mgr.NextPort(codeServerBasePort)
	if err := h.mgr.Start(*project.WorkspacePath, port); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Persist port to environment
	if _, err := h.envs.Update(r.Context(), env.ID, store.UpdateEnvironmentInput{
		CodeServerPort: &port,
	}); err != nil {
		// Non-fatal: process started but DB update failed
		writeJSON(w, http.StatusOK, map[string]any{
			"running": true,
			"port":    port,
			"url":     fmt.Sprintf("http://localhost:%d", port),
			"warning": "failed to persist port to database",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"running": true,
		"port":    port,
		"url":     fmt.Sprintf("http://localhost:%d", port),
	})
}

func (h *DevflowCodeServerHandler) handleStop(w http.ResponseWriter, r *http.Request) {
	_, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if env.CodeServerPort == nil || *env.CodeServerPort == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
		return
	}

	_ = h.mgr.Stop(*env.CodeServerPort)

	// Clear port in DB
	zeroPort := 0
	h.envs.Update(r.Context(), env.ID, store.UpdateEnvironmentInput{
		CodeServerPort: &zeroPort,
	})

	writeJSON(w, http.StatusOK, map[string]any{"running": false})
}

func (h *DevflowCodeServerHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	_, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	port := 0
	running := false
	if env.CodeServerPort != nil && *env.CodeServerPort > 0 {
		port = *env.CodeServerPort
		running = h.mgr.IsRunning(port)
	}

	resp := map[string]any{
		"running": running,
		"port":    port,
	}
	if running {
		resp["url"] = fmt.Sprintf("http://localhost:%d", port)
	}
	writeJSON(w, http.StatusOK, resp)
}
