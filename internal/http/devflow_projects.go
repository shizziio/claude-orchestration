package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/permissions"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowProjectsHandler handles project CRUD endpoints.
type DevflowProjectsHandler struct {
	projects store.ProjectStore
}

func NewDevflowProjectsHandler(projects store.ProjectStore) *DevflowProjectsHandler {
	return &DevflowProjectsHandler{projects: projects}
}

func (h *DevflowProjectsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/projects/{id}", requireAuth(permissions.RoleAdmin, h.handleDelete))
}

func (h *DevflowProjectsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if projects == nil {
		projects = []*store.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *DevflowProjectsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name              string     `json:"name"`
		Slug              string     `json:"slug"`
		Description       *string    `json:"description"`
		RepoURL           *string    `json:"repo_url"`
		DefaultBranch     string     `json:"default_branch"`
		GitCredentialID   *uuid.UUID `json:"git_credential_id"`
		DockerComposeFile *string    `json:"docker_compose_file"`
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
	userID := store.UserIDFromContext(r.Context())
	p, err := h.projects.Create(r.Context(), store.CreateProjectInput{
		Name:              body.Name,
		Slug:              body.Slug,
		Description:       body.Description,
		RepoURL:           body.RepoURL,
		DefaultBranch:     body.DefaultBranch,
		GitCredentialID:   body.GitCredentialID,
		DockerComposeFile: body.DockerComposeFile,
		CreatedBy:         userID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *DevflowProjectsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, p)
}

func (h *DevflowProjectsHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Name              *string    `json:"name"`
		Description       *string    `json:"description"`
		RepoURL           *string    `json:"repo_url"`
		DefaultBranch     *string    `json:"default_branch"`
		GitCredentialID   *uuid.UUID `json:"git_credential_id"`
		WorkspacePath     *string    `json:"workspace_path"`
		DockerComposeFile *string    `json:"docker_compose_file"`
		Status            *string    `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	p, err := h.projects.Update(r.Context(), id, store.UpdateProjectInput{
		Name:              body.Name,
		Description:       body.Description,
		RepoURL:           body.RepoURL,
		DefaultBranch:     body.DefaultBranch,
		GitCredentialID:   body.GitCredentialID,
		WorkspacePath:     body.WorkspacePath,
		DockerComposeFile: body.DockerComposeFile,
		Status:            body.Status,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *DevflowProjectsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.projects.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
