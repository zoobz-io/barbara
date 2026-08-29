//go:build testing

package stores

import (
	"testing"

	"github.com/zoobz-io/barbara/database/models"
)

// ClaimPending is a single UPDATE over the ids of a locking sub-select:
// UPDATE jobs SET status/attempts/updated_at WHERE id IN
// (SELECT id ... WHERE status=pending ORDER BY created_at ASC LIMIT n
// FOR UPDATE SKIP LOCKED) RETURNING. SKIP LOCKED is what lets many runners drain
// the outbox in parallel without blocking on each other.
func TestJobs_ClaimPending_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Jobs.ClaimPending(tenantCtx(), 20)

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "jobs" SET`,
		`"status" = ?`,
		`"attempts" = "attempts" + ?`,
		`"updated_at" = ?`,
		`"id" IN (SELECT "id" FROM "jobs"`,
		`"status" = ?`,
		`ORDER BY "created_at" ASC`,
		`LIMIT 20`,
		`FOR UPDATE SKIP LOCKED`,
		`RETURNING`,
	)
	wantArg(t, q, models.JobProcessing)
	wantArg(t, q, models.JobPending)
}

// MarkDone flips a single job to done by primary key.
func TestJobs_MarkDone_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_ = st.Jobs.MarkDone(tenantCtx(), "j-1")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "jobs" SET`,
		`"status" = ?`,
		`"updated_at" = ?`,
		`WHERE "id" = ?`,
	)
	wantArg(t, q, models.JobDone)
	wantArg(t, q, "j-1")
}

// MarkFailed records the error alongside the failed status, by primary key.
func TestJobs_MarkFailed_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_ = st.Jobs.MarkFailed(tenantCtx(), "j-1", "boom")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "jobs" SET`,
		`"last_error" = ?`,
		`"status" = ?`,
		`WHERE "id" = ?`,
	)
	wantArg(t, q, models.JobFailed)
	wantArg(t, q, "boom")
	wantArg(t, q, "j-1")
}
