package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowEnvironmentsHandler handles environment CRUD endpoints.
type DevflowEnvironmentsHandler struct {
	envs store.EnvironmentStore
}

func NewDevflowEnvironmentsHandler(envs store.EnvironmentStore) *DevflowEnvironmentsHandler {
	return &DevflowEnvironmentsHandler{envs: envs}
}

func (h *DevflowEnvironmentsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/environments", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/environments", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/environments/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{projectId}/environments/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/projects/{projectId}/environments/{id}", requireAuth("", h.handleDelete))
}

func (h *DevflowEnvironmentsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	envs, err := h.envs.ListByProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if envs == nil {
		envs = []*store.Environment{}
	}
	writeJSON(w, http.StatusOK, envs)
}

func (h *DevflowEnvironmentsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	var body struct {
		Name                  string  `json:"name"`
		Slug                  string  `json:"slug"`
		Branch                *string `json:"branch"`
		EnvVars               *string `json:"env_vars"` // KEY=VAL lines (plaintext; encrypted at rest)
		DockerComposeOverride *string `json:"docker_compose_override"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Name == "" || body.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug are required"})
		return
	}
	if !isValidSlug(body.Slug) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be lowercase alphanumeric with hyphens"})
		return
	}
	var envVarsBytes []byte
	if body.EnvVars != nil {
		envVarsBytes = []byte(*body.EnvVars)
	}
	env, err := h.envs.Create(r.Context(), store.CreateEnvironmentInput{
		ProjectID:             projectID,
		Name:                  body.Name,
		Slug:                  body.Slug,
		Branch:                body.Branch,
		EnvVars:               envVarsBytes,
		DockerComposeOverride: body.DockerComposeOverride,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, env)
}

func (h *DevflowEnvironmentsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	env, err := h.envs.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "environment not found"})
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (h *DevflowEnvironmentsHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Status                *string `json:"status"`
		Branch                *string `json:"branch"`
		EnvVars               *string `json:"env_vars"`
		DockerComposeOverride *string `json:"docker_compose_override"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	var envVarsBytes []byte
	if body.EnvVars != nil {
		envVarsBytes = []byte(*body.EnvVars)
	}
	env, err := h.envs.Update(r.Context(), id, store.UpdateEnvironmentInput{
		Status:                body.Status,
		Branch:                body.Branch,
		EnvVars:               envVarsBytes,
		DockerComposeOverride: body.DockerComposeOverride,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (h *DevflowEnvironmentsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.envs.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
