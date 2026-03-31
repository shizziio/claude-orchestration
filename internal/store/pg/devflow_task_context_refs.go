package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type PGTaskContextRefStore struct {
	db *sql.DB
}

func NewPGTaskContextRefStore(db *sql.DB) *PGTaskContextRefStore {
	return &PGTaskContextRefStore{db: db}
}

func (s *PGTaskContextRefStore) Attach(ctx context.Context, runID uuid.UUID, contextIDs []uuid.UUID) error {
	if len(contextIDs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, cid := range contextIDs {
		id := store.GenNewID()
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO ext_task_context_refs (id, run_id, task_context_id, created_at)
			 VALUES ($1,$2,$3,$4)
			 ON CONFLICT (run_id, task_context_id) DO NOTHING`,
			id, runID, cid, now,
		)
		if err != nil {
			return fmt.Errorf("attach context ref: %w", err)
		}
	}
	return nil
}

func (s *PGTaskContextRefStore) ListByRun(ctx context.Context, runID uuid.UUID) ([]*store.TaskContextRef, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, task_context_id, created_at
		 FROM ext_task_context_refs WHERE run_id = $1
		 ORDER BY created_at`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*store.TaskContextRef
	for rows.Next() {
		var r store.TaskContextRef
		if err := rows.Scan(&r.ID, &r.RunID, &r.TaskContextID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *PGTaskContextRefStore) Detach(ctx context.Context, runID, contextID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM ext_task_context_refs WHERE run_id = $1 AND task_context_id = $2`,
		runID, contextID,
	)
	return err
}
