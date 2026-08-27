// Package stores holds the shared data-access layer over the domain models.
// Stores are constructed once and shared across surfaces; multi-store writes
// with atomicity invariants live only as transactional methods here, never
// composed from individual store calls at call sites.
package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
)

// Jobs is the data-access layer for the OpenSearch-write outbox.
type Jobs struct {
	*sum.Database[models.Job]
}

// NewJobs creates a jobs store.
func NewJobs(db *sqlx.DB, renderer astql.Renderer) *Jobs {
	return &Jobs{
		Database: sum.NewDatabase[models.Job](db, "jobs", renderer),
	}
}

// Enqueue inserts a job inside the caller's transaction. Publishing calls this
// within the same transaction that moves the published pointer, so the pointer
// move and the outbox row commit atomically — no window where the pointer moved
// but no OS write is coming.
func (s *Jobs) Enqueue(ctx context.Context, tx *sqlx.Tx, j *models.Job) error {
	if err := s.SetTx(ctx, tx, "", j); err != nil {
		return fmt.Errorf("enqueuing job: %w", err)
	}
	return nil
}

// ClaimPending atomically claims up to limit pending jobs: it flips them to
// processing (bumping attempts) and returns them, skipping rows already locked
// by other claimers so concurrent runners don't collide.
//
// The claim is a single UPDATE over the ids of a locking sub-select —
// UPDATE ... WHERE id IN (SELECT id ... FOR UPDATE SKIP LOCKED) RETURNING — so
// the row is flipped and handed back in one statement, with no window where a
// row is selected but not yet marked. SKIP LOCKED is what lets many runners
// drain the outbox in parallel without blocking on each other.
func (s *Jobs) ClaimPending(ctx context.Context, limit int) ([]*models.Job, error) {
	pending := s.Query().
		Fields("id").
		Where("status", "=", "pending_status").
		OrderBy("created_at", "asc").
		Limit(limit).
		ForUpdate().
		SkipLocked()

	claimed, err := s.Modify().
		Set("status", "status").
		SetExpr("attempts", "+", "increment").
		Set("updated_at", "updated_at").
		WhereInSubquery("id", pending).
		ExecMany(ctx, map[string]any{
			"status":             models.JobProcessing,
			"increment":          1,
			"updated_at":         time.Now(),
			"sq1_pending_status": models.JobPending,
		})
	if err != nil {
		return nil, fmt.Errorf("claiming pending jobs: %w", err)
	}
	return claimed, nil
}

// MarkDone marks a job completed.
func (s *Jobs) MarkDone(ctx context.Context, id string) error {
	params := map[string]any{"id": id, "status": models.JobDone, "updated_at": time.Now()}
	_, err := s.Modify().
		Set("status", "status").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Exec(ctx, params)
	if err != nil {
		return fmt.Errorf("marking job done: %w", err)
	}
	return nil
}

// MarkFailed marks a job failed and records the error.
func (s *Jobs) MarkFailed(ctx context.Context, id, errMsg string) error {
	params := map[string]any{"id": id, "status": models.JobFailed, "last_error": errMsg, "updated_at": time.Now()}
	_, err := s.Modify().
		Set("status", "status").
		Set("last_error", "last_error").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Exec(ctx, params)
	if err != nil {
		return fmt.Errorf("marking job failed: %w", err)
	}
	return nil
}
