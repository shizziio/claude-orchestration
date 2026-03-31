package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/devflow/claudemd"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type DevflowProjectTeamsHandler struct {
	teams      store.ProjectTeamStore
	projects   store.ProjectStore
	projectCtx store.ProjectContextStore
}

func NewDevflowProjectTeamsHandler(teams store.ProjectTeamStore, projects store.ProjectStore, projectCtx store.ProjectContextStore) *DevflowProjectTeamsHandler {
	return &DevflowProjectTeamsHandler{teams: teams, projects: projects, projectCtx: projectCtx}
}

func (h *DevflowProjectTeamsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/teams", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/teams", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/teams/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{projectId}/teams/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/projects/{projectId}/teams/{id}", requireAuth("", h.handleDelete))
}

func (h *DevflowProjectTeamsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	teams, err := h.teams.ListByProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if teams == nil {
		teams = []*store.ProjectTeam{}
	}
	writeJSON(w, http.StatusOK, teams)
}

func (h *DevflowProjectTeamsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	var body struct {
		TeamName    string          `json:"team_name"`
		Description *string         `json:"description"`
		TeamConfig  json.RawMessage `json:"team_config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.TeamName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "team_name is required"})
		return
	}
	team, err := h.teams.Create(r.Context(), store.CreateProjectTeamInput{
		ProjectID:   projectID,
		TeamName:    body.TeamName,
		Description: body.Description,
		TeamConfig:  body.TeamConfig,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.regenClaudeMD(r, projectID)
	writeJSON(w, http.StatusCreated, team)
}

func (h *DevflowProjectTeamsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	team, err := h.teams.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, team)
}

func (h *DevflowProjectTeamsHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
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
		TeamName    *string         `json:"team_name"`
		Description *string         `json:"description"`
		TeamConfig  json.RawMessage `json:"team_config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	team, err := h.teams.Update(r.Context(), id, store.UpdateProjectTeamInput{
		TeamName:    body.TeamName,
		Description: body.Description,
		TeamConfig:  body.TeamConfig,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.regenClaudeMD(r, projectID)
	writeJSON(w, http.StatusOK, team)
}

func (h *DevflowProjectTeamsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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
	if err := h.teams.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.regenClaudeMD(r, projectID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *DevflowProjectTeamsHandler) regenClaudeMD(r *http.Request, projectID uuid.UUID) {
	project, err := h.projects.Get(r.Context(), projectID)
	if err != nil || project.WorkspacePath == nil || *project.WorkspacePath == "" {
		return
	}
	content, err := claudemd.Compose(r.Context(), projectID, h.projectCtx, h.teams)
	if err != nil || content == "" {
		return
	}
	_ = claudemd.WriteToWorkspace(content, *project.WorkspacePath)
}
