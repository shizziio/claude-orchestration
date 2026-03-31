package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/crypto"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// PGEnvironmentStore implements store.EnvironmentStore backed by Postgres.
type PGEnvironmentStore struct {
	db     *sql.DB
	encKey string
}

func NewPGEnvironmentStore(db *sql.DB, encryptionKey string) *PGEnvironmentStore {
	return &PGEnvironmentStore{db: db, encKey: encryptionKey}
}

const envColumns = `id, tenant_id, project_id, name, slug, status, compose_status, branch,
	docker_compose_override, port_bindings, code_server_port, preview_port, created_at, updated_at`

func scanEnvironment(row interface {
	Scan(...any) error
}) (*store.Environment, error) {
	var e store.Environment
	err := row.Scan(
		&e.ID, &e.TenantID, &e.ProjectID, &e.Name, &e.Slug, &e.Status, &e.ComposeStatus,
		&e.Branch, &e.DockerComposeOverride, &e.PortBindings,
		&e.CodeServerPort, &e.PreviewPort, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *PGEnvironmentStore) Create(ctx context.Context, in store.CreateEnvironmentInput) (*store.Environment, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	var envVarsEnc []byte
	if len(in.EnvVars) > 0 {
		enc, err := crypto.Encrypt(string(in.EnvVars), s.encKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt env vars: %w", err)
		}
		envVarsEnc = []byte(enc)
	}

	portBindings := in.PortBindings
	if len(portBindings) == 0 {
		portBindings = []byte("{}")
	}

	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_environments
		 (id, tenant_id, project_id, name, slug, status, branch, env_vars, docker_compose_override, port_bindings, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'dormant',$6,$7,$8,$9,$10,$11)`,
		id, tid, in.ProjectID, in.Name, in.Slug, in.Branch, envVarsEnc, in.DockerComposeOverride, portBindings, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGEnvironmentStore) Get(ctx context.Context, id uuid.UUID) (*store.Environment, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+envColumns+` FROM ext_environments WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	e, err := scanEnvironment(row)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	return e, nil
}

func (s *PGEnvironmentStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*store.Environment, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+envColumns+` FROM ext_environments WHERE tenant_id = $1 AND project_id = $2 ORDER BY created_at ASC`,
		tid, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.Environment
	for rows.Next() {
		e, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *PGEnvironmentStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateEnvironmentInput) (*store.Environment, error) {
	updates := map[string]any{}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.ComposeStatus != nil {
		updates["compose_status"] = *in.ComposeStatus
	}
	if in.Branch != nil {
		updates["branch"] = *in.Branch
	}
	if in.DockerComposeOverride != nil {
		updates["docker_compose_override"] = *in.DockerComposeOverride
	}
	if len(in.PortBindings) > 0 {
		updates["port_bindings"] = in.PortBindings
	}
	if in.CodeServerPort != nil {
		updates["code_server_port"] = *in.CodeServerPort
	}
	if in.PreviewPort != nil {
		updates["preview_port"] = *in.PreviewPort
	}
	if len(in.EnvVars) > 0 {
		enc, err := crypto.Encrypt(string(in.EnvVars), s.encKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt env vars: %w", err)
		}
		updates["env_vars"] = []byte(enc)
	}
	if err := execMapUpdate(ctx, s.db, "ext_environments", id, updates); err != nil {
		return nil, fmt.Errorf("update environment: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGEnvironmentStore) GetDecryptedEnvVars(ctx context.Context, id uuid.UUID) ([]byte, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	var enc []byte
	err = s.db.QueryRowContext(ctx,
		`SELECT env_vars FROM ext_environments WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	).Scan(&enc)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	if len(enc) == 0 || s.encKey == "" {
		return enc, nil
	}
	plain, err := crypto.Decrypt(string(enc), s.encKey)
	if err != nil {
		return nil, err
	}
	return []byte(plain), nil
}

func (s *PGEnvironmentStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_environments WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}
