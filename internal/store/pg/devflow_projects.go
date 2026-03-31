package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// PGProjectStore implements store.ProjectStore backed by Postgres.
type PGProjectStore struct {
	db *sql.DB
}

func NewPGProjectStore(db *sql.DB) *PGProjectStore {
	return &PGProjectStore{db: db}
}

const projectColumns = `id, tenant_id, name, slug, description, repo_url, default_branch,
	git_credential_id, workspace_path, docker_compose_file, status, settings, created_by, created_at, updated_at`

func scanProject(row interface {
	Scan(...any) error
}) (*store.Project, error) {
	var p store.Project
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Slug, &p.Description, &p.RepoURL, &p.DefaultBranch,
		&p.GitCredentialID, &p.WorkspacePath, &p.DockerComposeFile, &p.Status, &p.Settings,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *PGProjectStore) Create(ctx context.Context, in store.CreateProjectInput) (*store.Project, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	settings := in.Settings
	if len(settings) == 0 {
		settings = []byte("{}")
	}
	branch := in.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_projects
		 (id, tenant_id, name, slug, description, repo_url, default_branch, git_credential_id, docker_compose_file, settings, created_by, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'active',$12,$13)`,
		id, tid, in.Name, in.Slug, in.Description, in.RepoURL, branch, in.GitCredentialID,
		in.DockerComposeFile, settings, in.CreatedBy, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGProjectStore) Get(ctx context.Context, id uuid.UUID) (*store.Project, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM ext_projects WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	p, err := scanProject(row)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return p, nil
}

func (s *PGProjectStore) GetBySlug(ctx context.Context, slug string) (*store.Project, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+projectColumns+` FROM ext_projects WHERE tenant_id = $1 AND slug = $2`,
		tid, slug,
	)
	p, err := scanProject(row)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", slug)
	}
	return p, nil
}

func (s *PGProjectStore) List(ctx context.Context) ([]*store.Project, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+projectColumns+` FROM ext_projects WHERE tenant_id = $1 AND status = 'active' ORDER BY created_at DESC`,
		tid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *PGProjectStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateProjectInput) (*store.Project, error) {
	updates := map[string]any{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.RepoURL != nil {
		updates["repo_url"] = *in.RepoURL
	}
	if in.DefaultBranch != nil {
		updates["default_branch"] = *in.DefaultBranch
	}
	if in.GitCredentialID != nil {
		updates["git_credential_id"] = *in.GitCredentialID
	}
	if in.WorkspacePath != nil {
		updates["workspace_path"] = *in.WorkspacePath
	}
	if in.DockerComposeFile != nil {
		updates["docker_compose_file"] = *in.DockerComposeFile
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(in.Settings) > 0 {
		updates["settings"] = in.Settings
	}
	if err := execMapUpdate(ctx, s.db, "ext_projects", id, updates); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGProjectStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_projects WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}
