package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type DevflowTaskContextHandler struct {
	docs store.TaskContextStore
}

func NewDevflowTaskContextHandler(docs store.TaskContextStore) *DevflowTaskContextHandler {
	return &DevflowTaskContextHandler{docs: docs}
}

func (h *DevflowTaskContextHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/task-context", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/task-context", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/task-context/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{projectId}/task-context/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/projects/{projectId}/task-context/{id}", requireAuth("", h.handleDelete))
}

func (h *DevflowTaskContextHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	docs, err := h.docs.ListByProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if docs == nil {
		docs = []*store.TaskContext{}
	}
	writeJSON(w, http.StatusOK, docs)
}

func (h *DevflowTaskContextHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	var body struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Tags     []string `json:"tags"`
		FilePath *string  `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	doc, err := h.docs.Create(r.Context(), store.CreateTaskContextInput{
		ProjectID: projectID,
		Title:     body.Title,
		Content:   body.Content,
		Tags:      body.Tags,
		FilePath:  body.FilePath,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *DevflowTaskContextHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	doc, err := h.docs.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *DevflowTaskContextHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Title    *string  `json:"title"`
		Content  *string  `json:"content"`
		Tags     []string `json:"tags"`
		FilePath *string  `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	doc, err := h.docs.Update(r.Context(), id, store.UpdateTaskContextInput{
		Title:    body.Title,
		Content:  body.Content,
		Tags:     body.Tags,
		FilePath: body.FilePath,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *DevflowTaskContextHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.docs.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
