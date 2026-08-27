//go:build testing

// Integration tests for the jobs store, exercised against a real Postgres.
// Kept in-package (not testing/integration) so coverage is attributed to the
// store directly without cross-package -coverpkg. Skips when no database is
// reachable, so `make test` is a no-op on machines without the dev stack.
package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"

	_ "github.com/lib/pq"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// jobsDB connects to Postgres for integration tests, skipping when it is not
// reachable or the schema is absent — so the suite is a no-op without the dev
// stack and real only when it is up and migrated.
func jobsDB(t *testing.T) *sqlx.DB {
	t.Helper()
	sum.Reset() // fresh catalog per test — NewDatabase re-registers "db://jobs".
	sum.New()   // sum.NewDatabase requires the service singleton.
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("APP_DB_HOST", "127.0.0.1"), env("APP_DB_PORT", "5432"),
		env("APP_DB_USER", "barbara"), env("APP_DB_PASSWORD", "barbara"),
		env("APP_DB_NAME", "barbara"))

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres not reachable (%v); skipping integration test", err)
	}
	if _, err := db.Exec("DELETE FROM jobs"); err != nil {
		_ = db.Close()
		t.Skipf("jobs table not present (%v); skipping — run migrations first", err)
	}
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM jobs"); _ = db.Close() })
	return db
}

func TestJobs_EnqueueClaimMarkDone(t *testing.T) {
	db := jobsDB(t)
	store := NewJobs(db, astqlpg.New())
	ctx := context.Background()

	// Enqueue inside a transaction — the outbox contract.
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	j := &models.Job{
		ID:         "aaaaaaaa-0000-0000-0000-000000000001",
		TenantID:   "bbbbbbbb-0000-0000-0000-000000000001",
		DocumentID: "cccccccc-0000-0000-0000-000000000001",
		Operation:  models.JobIndex,
		Status:     models.JobPending,
		Payload:    models.JobPayload(`{"key":"guides/a.md","tenant_id":"t1"}`),
	}
	if err := store.Enqueue(ctx, tx, j); err != nil {
		_ = tx.Rollback()
		t.Fatalf("enqueue: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Claim it: status flips to processing, attempts bumped, payload round-trips.
	claimed, err := store.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed %d jobs, want 1", len(claimed))
	}
	got := claimed[0]
	if got.Status != models.JobProcessing {
		t.Errorf("status = %q, want processing", got.Status)
	}
	if got.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", got.Attempts)
	}
	// jsonb normalizes whitespace, so compare parsed values, not raw bytes.
	var gotPayload, wantPayload map[string]any
	if err := json.Unmarshal(got.Payload, &gotPayload); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	_ = json.Unmarshal([]byte(`{"key":"guides/a.md","tenant_id":"t1"}`), &wantPayload)
	if !reflect.DeepEqual(gotPayload, wantPayload) {
		t.Errorf("payload did not round-trip: got %v, want %v", gotPayload, wantPayload)
	}

	// A second claim returns nothing — the job is no longer pending.
	again, err := store.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("second claim returned %d, want 0", len(again))
	}

	if err := store.MarkDone(ctx, j.ID); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	final, err := store.Get(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if final.Status != models.JobDone {
		t.Errorf("final status = %q, want done", final.Status)
	}
}

func TestJobs_MarkFailedRecordsError(t *testing.T) {
	db := jobsDB(t)
	store := NewJobs(db, astqlpg.New())
	ctx := context.Background()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	j := &models.Job{
		ID:         "aaaaaaaa-0000-0000-0000-000000000002",
		TenantID:   "bbbbbbbb-0000-0000-0000-000000000001",
		DocumentID: "cccccccc-0000-0000-0000-000000000002",
		Operation:  models.JobDelete,
		Status:     models.JobPending,
	}
	if err := store.Enqueue(ctx, tx, j); err != nil {
		_ = tx.Rollback()
		t.Fatalf("enqueue: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if err := store.MarkFailed(ctx, j.ID, "opensearch unreachable"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	final, err := store.Get(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if final.Status != models.JobFailed {
		t.Errorf("status = %q, want failed", final.Status)
	}
	if final.LastError == nil || *final.LastError != "opensearch unreachable" {
		t.Errorf("last_error = %v, want recorded message", final.LastError)
	}
}
