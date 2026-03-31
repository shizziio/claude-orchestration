package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/devflow/compose"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowEnvLifecycleHandler manages environment start/stop/status/logs.
type DevflowEnvLifecycleHandler struct {
	envs     store.EnvironmentStore
	projects store.ProjectStore
}

func NewDevflowEnvLifecycleHandler(envs store.EnvironmentStore, projects store.ProjectStore) *DevflowEnvLifecycleHandler {
	return &DevflowEnvLifecycleHandler{envs: envs, projects: projects}
}

func (h *DevflowEnvLifecycleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/environments/{envId}/start", requireAuth("", h.handleStart))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/environments/{envId}/stop", requireAuth("", h.handleStop))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/environments/{envId}/status", requireAuth("", h.handleStatus))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/environments/{envId}/logs", requireAuth("", h.handleLogs))
}

func (h *DevflowEnvLifecycleHandler) resolveEnv(r *http.Request) (*store.Project, *store.Environment, error) {
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

func (h *DevflowEnvLifecycleHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	project, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Optional env vars from request body
	var body struct {
		EnvVars map[string]string `json:"env_vars"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	composeFile := ""
	if project.DockerComposeFile != nil {
		composeFile = *project.DockerComposeFile
	}
	overrideFile := ""
	if env.DockerComposeOverride != nil {
		overrideFile = *env.DockerComposeOverride
	}

	// Set compose_status = starting
	starting := "starting"
	h.envs.Update(r.Context(), env.ID, store.UpdateEnvironmentInput{ComposeStatus: &starting})

	slog.Info("devflow.compose.starting", "project", project.Slug, "env", env.Slug)

	if err := compose.Up(r.Context(), *project.WorkspacePath, composeFile, overrideFile, body.EnvVars); err != nil {
		errStatus := "error"
		h.envs.Update(r.Context(), env.ID, store.UpdateEnvironmentInput{ComposeStatus: &errStatus})
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	running := "running"
	runningStatus := "running"
	h.envs.Update(r.Context(), env.ID, store.UpdateEnvironmentInput{
		ComposeStatus: &running,
		Status:        &runningStatus,
	})

	slog.Info("devflow.compose.started", "project", project.Slug, "env", env.Slug)
	writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

func (h *DevflowEnvLifecycleHandler) handleStop(w http.ResponseWriter, r *http.Request) {
	project, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	composeFile := ""
	if project.DockerComposeFile != nil {
		composeFile = *project.DockerComposeFile
	}
	overrideFile := ""
	if env.DockerComposeOverride != nil {
		overrideFile = *env.DockerComposeOverride
	}

	if err := compose.Down(r.Context(), *project.WorkspacePath, composeFile, overrideFile); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	stopped := "stopped"
	dormant := "dormant"
	h.envs.Update(r.Context(), env.ID, store.UpdateEnvironmentInput{
		ComposeStatus: &stopped,
		Status:        &dormant,
	})

	slog.Info("devflow.compose.stopped", "project", project.Slug, "env", env.Slug)
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *DevflowEnvLifecycleHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	project, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	composeFile := ""
	if project.DockerComposeFile != nil {
		composeFile = *project.DockerComposeFile
	}
	overrideFile := ""
	if env.DockerComposeOverride != nil {
		overrideFile = *env.DockerComposeOverride
	}

	services, err := compose.Status(r.Context(), *project.WorkspacePath, composeFile, overrideFile)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if services == nil {
		services = []compose.ServiceStatus{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"compose_status": env.ComposeStatus,
		"services":       services,
	})
}

func (h *DevflowEnvLifecycleHandler) handleLogs(w http.ResponseWriter, r *http.Request) {
	project, env, err := h.resolveEnv(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	composeFile := ""
	if project.DockerComposeFile != nil {
		composeFile = *project.DockerComposeFile
	}
	overrideFile := ""
	if env.DockerComposeOverride != nil {
		overrideFile = *env.DockerComposeOverride
	}

	service := r.URL.Query().Get("service")
	tail := 100
	if t := r.URL.Query().Get("tail"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			tail = n
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, canFlush := w.(http.Flusher)

	if err := compose.Logs(r.Context(), *project.WorkspacePath, composeFile, overrideFile, service, tail, &sseWriter{w: w, f: flusher, canFlush: canFlush}); err != nil {
		// Client disconnected or compose logs ended — not an error
		return
	}
}

// sseWriter wraps http.ResponseWriter to write each chunk as an SSE data line.
type sseWriter struct {
	w        http.ResponseWriter
	f        http.Flusher
	canFlush bool
}

func (s *sseWriter) Write(p []byte) (int, error) {
	lines := string(p)
	for _, line := range splitLines(lines) {
		fmt.Fprintf(s.w, "data: %s\n\n", line)
	}
	if s.canFlush {
		s.f.Flush()
	}
	return len(p), nil
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
