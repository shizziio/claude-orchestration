package http

import (
	"database/sql"
	"net/http"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowDashboardHandler provides project stats and run retry.
type DevflowDashboardHandler struct {
	runs     store.DevflowRunStore
	projects store.ProjectStore
	db       *sql.DB
}

func NewDevflowDashboardHandler(runs store.DevflowRunStore, projects store.ProjectStore, db *sql.DB) *DevflowDashboardHandler {
	return &DevflowDashboardHandler{runs: runs, projects: projects, db: db}
}

func (h *DevflowDashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/stats", requireAuth("", h.handleStats))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/runs/{id}/retry", requireAuth("", h.handleRetry))
}

type projectStats struct {
	TotalRuns     int     `json:"total_runs"`
	CompletedRuns int     `json:"completed_runs"`
	FailedRuns    int     `json:"failed_runs"`
	RunningRuns   int     `json:"running_runs"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	AvgDurationMs int     `json:"avg_duration_ms"`
	SuccessRate   float64 `json:"success_rate"` // 0.0 - 1.0
}

func (h *DevflowDashboardHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}

	tid := store.TenantIDFromContext(r.Context())
	var stats projectStats
	err = h.db.QueryRowContext(r.Context(),
		`SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status = 'failed'),
			COUNT(*) FILTER (WHERE status = 'running'),
			COALESCE(SUM(cost_usd), 0),
			COALESCE(AVG(duration_ms) FILTER (WHERE duration_ms IS NOT NULL), 0)
		 FROM ext_devflow_runs
		 WHERE tenant_id = $1 AND project_id = $2`,
		tid, projectID,
	).Scan(&stats.TotalRuns, &stats.CompletedRuns, &stats.FailedRuns, &stats.RunningRuns, &stats.TotalCostUSD, &stats.AvgDurationMs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	finished := stats.CompletedRuns + stats.FailedRuns
	if finished > 0 {
		stats.SuccessRate = float64(stats.CompletedRuns) / float64(finished)
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *DevflowDashboardHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	runID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	// Get the original run
	original, err := h.runs.Get(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}
	if original.Status != "failed" && original.Status != "completed" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "can only retry completed or failed runs"})
		return
	}

	// Create a new run with same parameters
	userID := store.UserIDFromContext(r.Context())
	newRun, err := h.runs.Create(r.Context(), store.CreateRunInput{
		ProjectID:       projectID,
		EnvironmentID:   original.EnvironmentID,
		TaskDescription: original.TaskDescription,
		ContextPrompt:   original.ContextPrompt,
		Branch:          original.Branch,
		CreatedBy:       userID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, newRun)
}
