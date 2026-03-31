package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type PGProjectTeamStore struct {
	db *sql.DB
}

func NewPGProjectTeamStore(db *sql.DB) *PGProjectTeamStore {
	return &PGProjectTeamStore{db: db}
}

const projectTeamColumns = `id, tenant_id, project_id, team_name, description, team_config, created_at, updated_at`

func scanProjectTeam(row interface{ Scan(...any) error }) (*store.ProjectTeam, error) {
	var t store.ProjectTeam
	err := row.Scan(&t.ID, &t.TenantID, &t.ProjectID, &t.TeamName, &t.Description, &t.TeamConfig, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *PGProjectTeamStore) Create(ctx context.Context, in store.CreateProjectTeamInput) (*store.ProjectTeam, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	cfg := in.TeamConfig
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_project_teams (id, tenant_id, project_id, team_name, description, team_config, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, tid, in.ProjectID, in.TeamName, in.Description, cfg, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create project team: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGProjectTeamStore) Get(ctx context.Context, id uuid.UUID) (*store.ProjectTeam, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+projectTeamColumns+` FROM ext_project_teams WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	t, err := scanProjectTeam(row)
	if err != nil {
		return nil, fmt.Errorf("project team not found: %s", id)
	}
	return t, nil
}

func (s *PGProjectTeamStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*store.ProjectTeam, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+projectTeamColumns+` FROM ext_project_teams
		 WHERE tenant_id = $1 AND project_id = $2
		 ORDER BY created_at`,
		tid, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.ProjectTeam
	for rows.Next() {
		t, err := scanProjectTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *PGProjectTeamStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateProjectTeamInput) (*store.ProjectTeam, error) {
	updates := map[string]any{}
	if in.TeamName != nil {
		updates["team_name"] = *in.TeamName
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if len(in.TeamConfig) > 0 {
		updates["team_config"] = in.TeamConfig
	}
	if err := execMapUpdate(ctx, s.db, "ext_project_teams", id, updates); err != nil {
		return nil, fmt.Errorf("update project team: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGProjectTeamStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_project_teams WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}
