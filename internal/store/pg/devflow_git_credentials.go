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

// PGGitCredentialStore implements store.GitCredentialStore backed by Postgres.
type PGGitCredentialStore struct {
	db     *sql.DB
	encKey string
}

func NewPGGitCredentialStore(db *sql.DB, encryptionKey string) *PGGitCredentialStore {
	return &PGGitCredentialStore{db: db, encKey: encryptionKey}
}

func (s *PGGitCredentialStore) encryptBytes(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	enc, err := crypto.Encrypt(string(raw), s.encKey)
	if err != nil {
		return nil, err
	}
	return []byte(enc), nil
}

func (s *PGGitCredentialStore) decryptBytes(enc []byte) ([]byte, error) {
	if len(enc) == 0 {
		return nil, nil
	}
	plain, err := crypto.Decrypt(string(enc), s.encKey)
	if err != nil {
		return nil, err
	}
	return []byte(plain), nil
}

func (s *PGGitCredentialStore) Create(ctx context.Context, in store.CreateGitCredentialInput) (*store.GitCredential, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	privateKeyEnc, err := s.encryptBytes(in.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}
	tokenEnc, err := s.encryptBytes(in.Token)
	if err != nil {
		return nil, fmt.Errorf("encrypt token: %w", err)
	}

	now := time.Now().UTC()
	id := store.GenNewID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ext_git_credentials
		 (id, tenant_id, user_id, label, provider, host, auth_type, public_key, private_key, token, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		id, tid, in.UserID, in.Label, in.Provider, in.Host, in.AuthType, in.PublicKey, privateKeyEnc, tokenEnc, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create git credential: %w", err)
	}
	return &store.GitCredential{
		ID: id, TenantID: tid, UserID: in.UserID, Label: in.Label,
		Provider: in.Provider, Host: in.Host, AuthType: in.AuthType, PublicKey: in.PublicKey,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *PGGitCredentialStore) Get(ctx context.Context, id uuid.UUID) (*store.GitCredential, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	var c store.GitCredential
	err = s.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, user_id, label, provider, host, auth_type, public_key, created_at, updated_at
		 FROM ext_git_credentials WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	).Scan(&c.ID, &c.TenantID, &c.UserID, &c.Label, &c.Provider, &c.Host, &c.AuthType, &c.PublicKey, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("git credential not found: %s", id)
	}
	return &c, nil
}

func (s *PGGitCredentialStore) List(ctx context.Context, userID string) ([]*store.GitCredential, error) {
	tid, err := requireTenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, tenant_id, user_id, label, provider, host, auth_type, public_key, created_at, updated_at
		 FROM ext_git_credentials WHERE tenant_id = $1 AND user_id = $2 ORDER BY created_at DESC`,
		tid, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.GitCredential
	for rows.Next() {
		var c store.GitCredential
		if err := rows.Scan(&c.ID, &c.TenantID, &c.UserID, &c.Label, &c.Provider, &c.Host, &c.AuthType, &c.PublicKey, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *PGGitCredentialStore) Update(ctx context.Context, id uuid.UUID, in store.UpdateGitCredentialInput) (*store.GitCredential, error) {
	updates := map[string]any{}
	if in.Label != nil {
		updates["label"] = *in.Label
	}
	if in.Host != nil {
		updates["host"] = *in.Host
	}
	if in.PublicKey != nil {
		updates["public_key"] = *in.PublicKey
	}
	if len(in.PrivateKey) > 0 {
		enc, err := s.encryptBytes(in.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt private key: %w", err)
		}
		updates["private_key"] = enc
	}
	if len(in.Token) > 0 {
		enc, err := s.encryptBytes(in.Token)
		if err != nil {
			return nil, fmt.Errorf("encrypt token: %w", err)
		}
		updates["token"] = enc
	}
	if err := execMapUpdate(ctx, s.db, "ext_git_credentials", id, updates); err != nil {
		return nil, fmt.Errorf("update git credential: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PGGitCredentialStore) Delete(ctx context.Context, id uuid.UUID) error {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM ext_git_credentials WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	)
	return err
}

// GetPrivateKey returns the decrypted SSH private key.
// Used only when performing git operations — never exposed via API.
func (s *PGGitCredentialStore) GetPrivateKey(ctx context.Context, id uuid.UUID) ([]byte, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	var enc []byte
	err = s.db.QueryRowContext(ctx,
		`SELECT private_key FROM ext_git_credentials WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	).Scan(&enc)
	if err != nil {
		return nil, fmt.Errorf("git credential not found: %s", id)
	}
	return s.decryptBytes(enc)
}

// GetToken returns the decrypted PAT/token.
func (s *PGGitCredentialStore) GetToken(ctx context.Context, id uuid.UUID) ([]byte, error) {
	tClause, tArgs, err := tenantClauseN(ctx, 2)
	if err != nil {
		return nil, err
	}
	var enc []byte
	err = s.db.QueryRowContext(ctx,
		`SELECT token FROM ext_git_credentials WHERE id = $1`+tClause,
		append([]any{id}, tArgs...)...,
	).Scan(&enc)
	if err != nil {
		return nil, fmt.Errorf("git credential not found: %s", id)
	}
	return s.decryptBytes(enc)
}
