package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type PGDevflowWebhookStore struct {
	db *sql.DB
}

func NewPGDevflowWebhookStore(db *sql.DB) *PGDevflowWebhookStore {
	return &PGDevflowWebhookStore{db: db}
}

const webhookColumns = `id, tenant_id, project_id, event_type, branch_filter, task_template, enabled, secret, created_at, updated_at`

func scanWebhook(row interface{ Scan(...any) error }) (*store.DevflowWebhook, error) {
	var w store.DevflowWebhook
	err := row.Scan(&w.ID, &w.TenantID, &w.ProjectID, &w.EventType, &w.BranchFilter, &w.TaskTemplate, &w.Enabled, &w.Secret, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *PGDevflowWebhookStore) Create(ctx context.Context, in store.CreateWebhookInput) (*store.DevflowWebhook, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	eventType := in.EventType
	if eventType == "" {
		eventType = "push"
	}
	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_devflow_webhooks (id, tenant_id, project_id, event_type, branch_filter, task_template, enabled, secret, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,true,$7,$8,$9)`,
		id, tid, in.ProjectID, eventType, in.BranchFilter, in.TaskTemplate, in.Secret, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGDevflowWebhookStore) Get(ctx context.Context, id uuid.UUID) (*store.DevflowWebhook, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+webhookColumns+` FROM ext_devflow_webhooks WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return scanWebhook(row)
}

func (s *PGDevflowWebhookStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*store.DevflowWebhook, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+webhookColumns+` FROM ext_devflow_webhooks WHERE tenant_id = $1 AND project_id = $2 ORDER BY created_at`,
		tid, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.DevflowWebhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *PGDevflowWebhookStore) ListEnabled(ctx context.Context, projectID uuid.UUID, eventType string) ([]*store.DevflowWebhook, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+webhookColumns+` FROM ext_devflow_webhooks
		 WHERE tenant_id = $1 AND project_id = $2 AND event_type = $3 AND enabled = true
		 ORDER BY created_at`,
		tid, projectID, eventType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.DevflowWebhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *PGDevflowWebhookStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateWebhookInput) (*store.DevflowWebhook, error) {
	updates := map[string]any{}
	if in.EventType != nil {
		updates["event_type"] = *in.EventType
	}
	if in.BranchFilter != nil {
		updates["branch_filter"] = *in.BranchFilter
	}
	if in.TaskTemplate != nil {
		updates["task_template"] = *in.TaskTemplate
	}
	if in.Enabled != nil {
		updates["enabled"] = *in.Enabled
	}
	if in.Secret != nil {
		updates["secret"] = *in.Secret
	}
	if err := execMapUpdate(ctx, s.db, "ext_devflow_webhooks", id, updates); err != nil {
		return nil, fmt.Errorf("update webhook: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGDevflowWebhookStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_devflow_webhooks WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}
