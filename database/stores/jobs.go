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
	db *sqlx.DB
}

// NewJobs creates a jobs store. The raw *sqlx.DB is retained for the atomic
// claim query, which needs FOR UPDATE SKIP LOCKED that the builder can't express.
func NewJobs(db *sqlx.DB, renderer astql.Renderer) *Jobs {
	return &Jobs{
		Database: sum.NewDatabase[models.Job](db, "jobs", renderer),
		db:       db,
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

// claimQuery atomically claims up to limit pending jobs: it flips them to
// processing (bumping attempts) and returns them, skipping rows locked by other
// claimers so concurrent runners don't collide.
const claimQuery = `
UPDATE jobs SET status = $1, attempts = attempts + 1, updated_at = now()
WHERE id IN (
    SELECT id FROM jobs
    WHERE status = $2
    ORDER BY created_at
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
RETURNING id, tenant_id, document_id, operation, payload, status, attempts, last_error, created_at, updated_at`

// ClaimPending claims up to limit pending jobs, marking them processing.
func (s *Jobs) ClaimPending(ctx context.Context, limit int) ([]*models.Job, error) {
	rows, err := s.db.QueryxContext(ctx, claimQuery, models.JobProcessing, models.JobPending, limit)
	if err != nil {
		return nil, fmt.Errorf("claiming pending jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var jobs []*models.Job
	for rows.Next() {
		var j models.Job
		if scanErr := rows.StructScan(&j); scanErr != nil {
			return nil, fmt.Errorf("scanning job: %w", scanErr)
		}
		jobs = append(jobs, &j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating claimed jobs: %w", err)
	}
	return jobs, nil
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
