package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type PGTaskContextStore struct {
	db *sql.DB
}

func NewPGTaskContextStore(db *sql.DB) *PGTaskContextStore {
	return &PGTaskContextStore{db: db}
}

const taskContextColumns = `id, tenant_id, project_id, title, content, tags, file_path, created_at, updated_at`

func scanTaskContext(row interface{ Scan(...any) error }) (*store.TaskContext, error) {
	var c store.TaskContext
	err := row.Scan(&c.ID, &c.TenantID, &c.ProjectID, &c.Title, &c.Content, pq.Array(&c.Tags), &c.FilePath, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return &c, nil
}

func (s *PGTaskContextStore) Create(ctx context.Context, in store.CreateTaskContextInput) (*store.TaskContext, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_task_context (id, tenant_id, project_id, title, content, tags, file_path, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, tid, in.ProjectID, in.Title, in.Content, pq.Array(tags), in.FilePath, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create task context: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGTaskContextStore) Get(ctx context.Context, id uuid.UUID) (*store.TaskContext, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+taskContextColumns+` FROM ext_task_context WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	c, err := scanTaskContext(row)
	if err != nil {
		return nil, fmt.Errorf("task context not found: %s", id)
	}
	return c, nil
}

func (s *PGTaskContextStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*store.TaskContext, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskContextColumns+` FROM ext_task_context
		 WHERE tenant_id = $1 AND project_id = $2
		 ORDER BY created_at DESC`,
		tid, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.TaskContext
	for rows.Next() {
		c, err := scanTaskContext(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PGTaskContextStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateTaskContextInput) (*store.TaskContext, error) {
	updates := map[string]any{}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Content != nil {
		updates["content"] = *in.Content
	}
	if in.Tags != nil {
		updates["tags"] = pq.Array(in.Tags)
	}
	if in.FilePath != nil {
		updates["file_path"] = *in.FilePath
	}
	if err := execMapUpdate(ctx, s.db, "ext_task_context", id, updates); err != nil {
		return nil, fmt.Errorf("update task context: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGTaskContextStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_task_context WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}
