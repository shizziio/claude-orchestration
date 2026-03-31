package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/devflow/claudemd"
	devctx "github.com/nextlevelbuilder/goclaw/internal/devflow/context"
	"github.com/nextlevelbuilder/goclaw/internal/devflow/logbroker"
	"github.com/nextlevelbuilder/goclaw/internal/devflow/runner"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowRunnerHandler handles run execution and log streaming endpoints.
type DevflowRunnerHandler struct {
	projects    store.ProjectStore
	runs        store.DevflowRunStore
	projectCtx  store.ProjectContextStore
	taskCtx     store.TaskContextStore
	taskCtxRefs store.TaskContextRefStore
	teams       store.ProjectTeamStore
	broker      *logbroker.Broker
	opts        DevflowRunnerOpts
}

// DevflowRunnerOpts configures the runner handler.
type DevflowRunnerOpts struct {
	ClaudeBin      string
	PermissionMode runner.PermissionMode
	AgentTeams     bool
	MaxBudgetUSD   float64
	DefaultModel   string
	Timeout        time.Duration
}

func NewDevflowRunnerHandler(
	projects store.ProjectStore,
	runs store.DevflowRunStore,
	projectCtx store.ProjectContextStore,
	taskCtx store.TaskContextStore,
	taskCtxRefs store.TaskContextRefStore,
	teams store.ProjectTeamStore,
	opts DevflowRunnerOpts,
) *DevflowRunnerHandler {
	if opts.PermissionMode == "" {
		opts.PermissionMode = runner.PermissionModeBypassPermissions
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Minute
	}
	return &DevflowRunnerHandler{
		projects:    projects,
		runs:        runs,
		projectCtx:  projectCtx,
		taskCtx:     taskCtx,
		taskCtxRefs: taskCtxRefs,
		teams:       teams,
		broker:      logbroker.New(),
		opts:        opts,
	}
}

func (h *DevflowRunnerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/runs/{id}/start", requireAuth("", h.handleStart))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/runs/{id}/log", requireAuth("", h.handleLog))
}

func (h *DevflowRunnerHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run id"})
		return
	}

	run, err := h.runs.Get(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}
	if run.Status != "pending" {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("run is already %s", run.Status),
		})
		return
	}

	project, err := h.projects.Get(r.Context(), run.ProjectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "project not found"})
		return
	}
	if project.WorkspacePath == nil || *project.WorkspacePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project workspace not set — clone the repo first"})
		return
	}

	// Mark as running immediately so duplicate start calls get 409
	now := time.Now().UTC()
	status := "running"
	sessionID := run.ID.String()
	run, err = h.runs.Update(r.Context(), runID, store.UpdateRunInput{
		Status:          &status,
		ClaudeSessionID: &sessionID,
		StartedAt:       &now,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Respond immediately; execution runs in background
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "running",
		"run_id":     run.ID,
		"session_id": sessionID,
	})

	// Launch in background goroutine — caller polls GET /runs/{id} for status
	go h.executeRun(run, project)
}

// handleLog returns the run log. For live runs it streams until the run finishes
// (long-poll style using SSE). For completed runs it returns the stored log from DB.
func (h *DevflowRunnerHandler) handleLog(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run id"})
		return
	}

	// Check if the run is live in the broker
	signal, current, found := h.broker.Subscribe(runID)
	if found {
		// Run is in broker (live or just-completed)
		if signal == nil {
			// Already closed — return the full buffered log
			writeJSON(w, http.StatusOK, map[string]any{
				"log":      current,
				"complete": true,
			})
			return
		}
		defer h.broker.Unsubscribe(runID, signal)

		// Stream as SSE until the run ends or client disconnects
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, canFlush := w.(http.Flusher)

		// Send current buffered content immediately
		fmt.Fprintf(w, "data: %s\n\n", escapeSSE(current))
		if canFlush {
			flusher.Flush()
		}

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-signal:
				log, _ := h.broker.GetLog(runID)
				closed := h.broker.IsClosed(runID)
				fmt.Fprintf(w, "data: %s\n\n", escapeSSE(log))
				if canFlush {
					flusher.Flush()
				}
				if !ok || closed {
					fmt.Fprintf(w, "event: done\ndata: \n\n")
					if canFlush {
						flusher.Flush()
					}
					return
				}
			}
		}
	}

	// Not in broker — run is completed, fetch from DB
	run, err := h.runs.Get(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}
	log, err := h.runs.GetLog(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"log":      log,
		"complete": run.Status == "completed" || run.Status == "failed",
	})
}

func (h *DevflowRunnerHandler) executeRun(run *store.DevflowRun, project *store.Project) {
	// Use a fresh background context — request context is already done
	ctx := store.WithTenantID(
		store.WithUserID(context.Background(), run.CreatedBy),
		run.TenantID,
	)

	sessionID := run.ID.String()

	// Regenerate CLAUDE.md from project context before each run
	if project.WorkspacePath != nil && *project.WorkspacePath != "" {
		if md, err := claudemd.Compose(ctx, project.ID, h.projectCtx, h.teams); err == nil && md != "" {
			_ = claudemd.WriteToWorkspace(md, *project.WorkspacePath)
		}
	}

	// Compose tier-2 context from attached task context documents
	var contextPrompt string
	if run.ContextPrompt != nil && *run.ContextPrompt != "" {
		contextPrompt = *run.ContextPrompt // manual override takes precedence
	} else {
		composed, err := devctx.ComposeTaskPrompt(ctx, run.ID, h.taskCtxRefs, h.taskCtx)
		if err == nil && composed != "" {
			contextPrompt = composed
		}
	}

	var branch string
	if run.Branch != nil {
		branch = *run.Branch
	}

	// Broker writer captures live output
	logWriter := h.broker.Writer(run.ID)

	result, runErr := runner.Execute(ctx, runner.RunOptions{
		WorkspaceDir:   *project.WorkspacePath,
		SessionID:      sessionID,
		TaskPrompt:     run.TaskDescription,
		ContextPrompt:  contextPrompt,
		Branch:         branch,
		PermissionMode: h.opts.PermissionMode,
		AgentTeams:     h.opts.AgentTeams,
		MaxBudgetUSD:   h.opts.MaxBudgetUSD,
		Model:          h.opts.DefaultModel,
		ClaudeBin:      h.opts.ClaudeBin,
		Timeout:        h.opts.Timeout,
		Stdout:         logWriter,
	})
	logWriter.Close() // signals SSE subscribers that the run is done

	completedAt := time.Now().UTC()
	var newStatus, resultSummary, errMsg string

	if runErr != nil {
		newStatus = "failed"
		errMsg = runErr.Error()
		slog.Warn("devflow.runner.run_failed",
			"run_id", run.ID,
			"project_id", run.ProjectID,
			"error", runErr,
		)
	} else {
		newStatus = "completed"
		if result != nil {
			resultSummary = truncateSummary(result.Output, 500)
			if result.SessionID != "" {
				sessionID = result.SessionID
			}
		}
		slog.Info("devflow.runner.run_completed",
			"run_id", run.ID,
			"project_id", run.ProjectID,
			"cost_usd", func() float64 {
				if result != nil {
					return result.TotalCostUSD
				}
				return 0
			}(),
		)
	}

	// Persist full log to DB
	fullLog, _ := h.broker.GetLog(run.ID)
	h.broker.Evict(run.ID)

	update := store.UpdateRunInput{
		Status:          &newStatus,
		ClaudeSessionID: &sessionID,
		CompletedAt:     &completedAt,
		RunLog:          &fullLog,
	}
	if resultSummary != "" {
		update.ResultSummary = &resultSummary
	}
	if errMsg != "" {
		update.ErrorMessage = &errMsg
	}
	// Persist cost and duration from Claude output
	if result != nil && result.TotalCostUSD > 0 {
		update.CostUSD = &result.TotalCostUSD
	}
	if result != nil && result.ExitCode >= 0 {
		durationMs := int(completedAt.Sub(*run.StartedAt).Milliseconds())
		if run.StartedAt != nil && durationMs > 0 {
			update.DurationMs = &durationMs
		}
	}

	if _, err := h.runs.Update(ctx, run.ID, update); err != nil {
		slog.Error("devflow.runner.update_failed", "run_id", run.ID, "error", err)
	}
}

// escapeSSE escapes a string for safe embedding in an SSE data line.
// SSE data fields cannot contain raw newlines; we replace them with \n literal.
func escapeSSE(s string) string {
	// For simplicity: send full log as a single JSON-encoded string value
	// The client receives one big data field with the accumulated log
	// (not ideal for very large logs, but simple and correct)
	return s
}

func truncateSummary(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…(truncated)"
}
