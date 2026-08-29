//go:build testing

package stores

import (
	"testing"

	"github.com/zoobz-io/barbara/database/models"
)

// Publish first loads the version to be published, scoped to the request's
// tenant (id AND tenant_id) — a version from another tenant is never visible.
func TestPublish_ValidatesVersionInTenant(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Publish(tenantCtx(), "d-1", "v-1")

	// The tenant-scoped version lookup is the first query; the mock returns no
	// row, so the pointer move never runs and this is the captured query.
	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "versions"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "v-1")
	wantArg(t, q, testTenant)
}

// Unpublish clears the published pointer (published_version_id = NULL) and bumps
// updated_at, scoped to id + tenant, returning the updated row.
func TestUnpublish_ClearsPointer(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Unpublish(tenantCtx(), "d-1")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "documents" SET`,
		`"published_version_id" = ?`,
		`"updated_at" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`RETURNING`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

// Publish's full transactional write, driven to completion: after validating the
// version and loading the document, it moves the published pointer to the
// version and enqueues an index job in the SAME transaction — so the pointer
// move and the outbox row commit atomically.
func TestPublish_MovesPointerAndEnqueuesIndexJob(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(versionRow()) // Versions.Get (belongs to d-1)
	cfg.PushRowData(docRow(nil))  // Documents.Get
	cfg.PushRowData(docRow(nil))  // setPublishedVersion RETURNING
	cfg.PushRowData(jobRow())     // Jobs.Enqueue RETURNING
	_, _ = st.Publish(tenantCtx(), "d-1", "v-1")

	// The pointer move: published_version_id set to the version, tenant-scoped.
	move := queryAt(t, capture, 2)
	wantSQL(t, move,
		`UPDATE "documents" SET`,
		`"published_version_id" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
	)
	wantArg(t, move, "v-1") // pointer set to the published version

	// The outbox insert: an index job, upserted by id (ON CONFLICT), pending.
	enqueue := queryAt(t, capture, 3)
	wantSQL(t, enqueue,
		`INSERT INTO "jobs"`,
		`ON CONFLICT`,
		`"operation"`,
		`"status"`,
		`"payload"`,
	)
	wantArg(t, enqueue, models.JobIndex)
	wantArg(t, enqueue, models.JobPending)
}
