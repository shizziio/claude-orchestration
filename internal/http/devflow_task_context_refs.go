package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type DevflowTaskContextRefsHandler struct {
	refs store.TaskContextRefStore
}

func NewDevflowTaskContextRefsHandler(refs store.TaskContextRefStore) *DevflowTaskContextRefsHandler {
	return &DevflowTaskContextRefsHandler{refs: refs}
}

func (h *DevflowTaskContextRefsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/runs/{runId}/context-refs", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/runs/{runId}/context-refs", requireAuth("", h.handleAttach))
	mux.HandleFunc("DELETE /v1/devflow/projects/{projectId}/runs/{runId}/context-refs/{contextId}", requireAuth("", h.handleDetach))
}

func (h *DevflowTaskContextRefsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(r.PathValue("runId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runId"})
		return
	}
	refs, err := h.refs.ListByRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if refs == nil {
		refs = []*store.TaskContextRef{}
	}
	writeJSON(w, http.StatusOK, refs)
}

func (h *DevflowTaskContextRefsHandler) handleAttach(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(r.PathValue("runId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runId"})
		return
	}
	var body struct {
		ContextIDs []uuid.UUID `json:"context_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := h.refs.Attach(r.Context(), runID, body.ContextIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *DevflowTaskContextRefsHandler) handleDetach(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(r.PathValue("runId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runId"})
		return
	}
	contextID, err := uuid.Parse(r.PathValue("contextId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid contextId"})
		return
	}
	if err := h.refs.Detach(r.Context(), runID, contextID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
