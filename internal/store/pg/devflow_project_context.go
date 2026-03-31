package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type PGProjectContextStore struct {
	db *sql.DB
}

func NewPGProjectContextStore(db *sql.DB) *PGProjectContextStore {
	return &PGProjectContextStore{db: db}
}

const projectContextColumns = `id, tenant_id, project_id, doc_type, title, content, sort_order, created_at, updated_at`

func scanProjectContext(row interface{ Scan(...any) error }) (*store.ProjectContext, error) {
	var c store.ProjectContext
	err := row.Scan(&c.ID, &c.TenantID, &c.ProjectID, &c.DocType, &c.Title, &c.Content, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *PGProjectContextStore) Create(ctx context.Context, in store.CreateProjectContextInput) (*store.ProjectContext, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	docType := in.DocType
	if docType == "" {
		docType = "rules"
	}
	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_project_context (id, tenant_id, project_id, doc_type, title, content, sort_order, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, tid, in.ProjectID, docType, in.Title, in.Content, in.SortOrder, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create project context: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGProjectContextStore) Get(ctx context.Context, id uuid.UUID) (*store.ProjectContext, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+projectContextColumns+` FROM ext_project_context WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	c, err := scanProjectContext(row)
	if err != nil {
		return nil, fmt.Errorf("project context not found: %s", id)
	}
	return c, nil
}

func (s *PGProjectContextStore) ListByProject(ctx context.Context, projectID uuid.UUID, docType string) ([]*store.ProjectContext, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if docType != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT `+projectContextColumns+` FROM ext_project_context
			 WHERE tenant_id = $1 AND project_id = $2 AND doc_type = $3
			 ORDER BY sort_order, created_at`,
			tid, projectID, docType,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT `+projectContextColumns+` FROM ext_project_context
			 WHERE tenant_id = $1 AND project_id = $2
			 ORDER BY doc_type, sort_order, created_at`,
			tid, projectID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.ProjectContext
	for rows.Next() {
		c, err := scanProjectContext(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PGProjectContextStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateProjectContextInput) (*store.ProjectContext, error) {
	updates := map[string]any{}
	if in.DocType != nil {
		updates["doc_type"] = *in.DocType
	}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Content != nil {
		updates["content"] = *in.Content
	}
	if in.SortOrder != nil {
		updates["sort_order"] = *in.SortOrder
	}
	if err := execMapUpdate(ctx, s.db, "ext_project_context", id, updates); err != nil {
		return nil, fmt.Errorf("update project context: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGProjectContextStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_project_context WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}
