package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// PGDevflowRunStore implements store.DevflowRunStore backed by Postgres.
type PGDevflowRunStore struct {
	db *sql.DB
}

func NewPGDevflowRunStore(db *sql.DB) *PGDevflowRunStore {
	return &PGDevflowRunStore{db: db}
}

const runColumns = `id, tenant_id, project_id, environment_id, task_description, context_prompt,
	branch, status, claude_session_id, result_summary, error_message, run_log,
	cost_usd, duration_ms, created_by, started_at, completed_at, created_at, updated_at`

func scanRun(row interface {
	Scan(...any) error
}) (*store.DevflowRun, error) {
	var r store.DevflowRun
	err := row.Scan(
		&r.ID, &r.TenantID, &r.ProjectID, &r.EnvironmentID, &r.TaskDescription, &r.ContextPrompt,
		&r.Branch, &r.Status, &r.ClaudeSessionID, &r.ResultSummary, &r.ErrorMessage, &r.RunLog,
		&r.CostUSD, &r.DurationMs, &r.CreatedBy,
		&r.StartedAt, &r.CompletedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *PGDevflowRunStore) Create(ctx context.Context, in store.CreateRunInput) (*store.DevflowRun, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_devflow_runs
		 (id, tenant_id, project_id, environment_id, task_description, context_prompt, branch, status, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8,$9,$10)`,
		id, tid, in.ProjectID, in.EnvironmentID, in.TaskDescription, in.ContextPrompt, in.Branch, in.CreatedBy, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create devflow run: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGDevflowRunStore) Get(ctx context.Context, id uuid.UUID) (*store.DevflowRun, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+runColumns+` FROM ext_devflow_runs WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	r, err := scanRun(row)
	if err != nil {
		return nil, fmt.Errorf("run not found: %s", id)
	}
	return r, nil
}

func (s *PGDevflowRunStore) ListByProject(ctx context.Context, projectID uuid.UUID, limit int) ([]*store.DevflowRun, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+runColumns+` FROM ext_devflow_runs
		 WHERE tenant_id = $1 AND project_id = $2
		 ORDER BY created_at DESC LIMIT $3`,
		tid, projectID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.DevflowRun
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *PGDevflowRunStore) GetLog(ctx context.Context, id uuid.UUID) (string, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return "", err
	}
	var log sql.NullString
	row := s.db.QueryRowContext(ctx,
		`SELECT run_log FROM ext_devflow_runs WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	if err := row.Scan(&log); err != nil {
		return "", fmt.Errorf("get run log: %w", err)
	}
	return log.String, nil
}

func (s *PGDevflowRunStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateRunInput) (*store.DevflowRun, error) {
	updates := map[string]any{}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.ClaudeSessionID != nil {
		updates["claude_session_id"] = *in.ClaudeSessionID
	}
	if in.ResultSummary != nil {
		updates["result_summary"] = *in.ResultSummary
	}
	if in.ErrorMessage != nil {
		updates["error_message"] = *in.ErrorMessage
	}
	if in.RunLog != nil {
		updates["run_log"] = *in.RunLog
	}
	if in.CostUSD != nil {
		updates["cost_usd"] = *in.CostUSD
	}
	if in.DurationMs != nil {
		updates["duration_ms"] = *in.DurationMs
	}
	if in.StartedAt != nil {
		updates["started_at"] = *in.StartedAt
	}
	if in.CompletedAt != nil {
		updates["completed_at"] = *in.CompletedAt
	}
	if err := execMapUpdate(ctx, s.db, "ext_devflow_runs", id, updates); err != nil {
		return nil, fmt.Errorf("update run: %w", err)
	}
	return s.Get(ctx, id)
}
