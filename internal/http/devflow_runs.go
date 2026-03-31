package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowRunsHandler handles devflow run tracking endpoints.
type DevflowRunsHandler struct {
	runs store.DevflowRunStore
}

func NewDevflowRunsHandler(runs store.DevflowRunStore) *DevflowRunsHandler {
	return &DevflowRunsHandler{runs: runs}
}

func (h *DevflowRunsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/runs", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/runs", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/runs/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{projectId}/runs/{id}", requireAuth("", h.handleUpdate))
}

func (h *DevflowRunsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	runs, err := h.runs.ListByProject(r.Context(), projectID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if runs == nil {
		runs = []*store.DevflowRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (h *DevflowRunsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	var body struct {
		EnvironmentID   *uuid.UUID `json:"environment_id"`
		TaskDescription string     `json:"task_description"`
		ContextPrompt   *string    `json:"context_prompt"`
		Branch          *string    `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.TaskDescription == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task_description is required"})
		return
	}
	userID := store.UserIDFromContext(r.Context())
	run, err := h.runs.Create(r.Context(), store.CreateRunInput{
		ProjectID:       projectID,
		EnvironmentID:   body.EnvironmentID,
		TaskDescription: body.TaskDescription,
		ContextPrompt:   body.ContextPrompt,
		Branch:          body.Branch,
		CreatedBy:       userID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (h *DevflowRunsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	run, err := h.runs.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *DevflowRunsHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Status          *string `json:"status"`
		ClaudeSessionID *string `json:"claude_session_id"`
		ResultSummary   *string `json:"result_summary"`
		ErrorMessage    *string `json:"error_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	run, err := h.runs.Update(r.Context(), id, store.UpdateRunInput{
		Status:          body.Status,
		ClaudeSessionID: body.ClaudeSessionID,
		ResultSummary:   body.ResultSummary,
		ErrorMessage:    body.ErrorMessage,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}
