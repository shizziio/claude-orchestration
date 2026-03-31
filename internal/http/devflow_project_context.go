package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/devflow/claudemd"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type DevflowProjectContextHandler struct {
	contexts store.ProjectContextStore
	projects store.ProjectStore
	teams    store.ProjectTeamStore
}

func NewDevflowProjectContextHandler(contexts store.ProjectContextStore, projects store.ProjectStore, teams store.ProjectTeamStore) *DevflowProjectContextHandler {
	return &DevflowProjectContextHandler{contexts: contexts, projects: projects, teams: teams}
}

func (h *DevflowProjectContextHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/context", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/context", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/context/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{projectId}/context/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/projects/{projectId}/context/{id}", requireAuth("", h.handleDelete))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/claude-md/preview", requireAuth("", h.handlePreview))
}

func (h *DevflowProjectContextHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	docType := r.URL.Query().Get("doc_type")
	entries, err := h.contexts.ListByProject(r.Context(), projectID, docType)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []*store.ProjectContext{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *DevflowProjectContextHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	var body struct {
		DocType   string `json:"doc_type"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	entry, err := h.contexts.Create(r.Context(), store.CreateProjectContextInput{
		ProjectID: projectID,
		DocType:   body.DocType,
		Title:     body.Title,
		Content:   body.Content,
		SortOrder: body.SortOrder,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.regenClaudeMD(r, projectID)
	writeJSON(w, http.StatusCreated, entry)
}

func (h *DevflowProjectContextHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	entry, err := h.contexts.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *DevflowProjectContextHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		DocType   *string `json:"doc_type"`
		Title     *string `json:"title"`
		Content   *string `json:"content"`
		SortOrder *int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	entry, err := h.contexts.Update(r.Context(), id, store.UpdateProjectContextInput{
		DocType:   body.DocType,
		Title:     body.Title,
		Content:   body.Content,
		SortOrder: body.SortOrder,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.regenClaudeMD(r, projectID)
	writeJSON(w, http.StatusOK, entry)
}

func (h *DevflowProjectContextHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.contexts.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.regenClaudeMD(r, projectID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *DevflowProjectContextHandler) handlePreview(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	content, err := claudemd.Compose(r.Context(), projectID, h.contexts, h.teams)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

// regenClaudeMD regenerates CLAUDE.md in the project workspace (best-effort).
func (h *DevflowProjectContextHandler) regenClaudeMD(r *http.Request, projectID uuid.UUID) {
	project, err := h.projects.Get(r.Context(), projectID)
	if err != nil || project.WorkspacePath == nil || *project.WorkspacePath == "" {
		return
	}
	content, err := claudemd.Compose(r.Context(), projectID, h.contexts, h.teams)
	if err != nil || content == "" {
		return
	}
	_ = claudemd.WriteToWorkspace(content, *project.WorkspacePath)
}
