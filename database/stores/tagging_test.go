//go:build testing

package stores

import (
	"testing"

	"github.com/zoobz-io/barbara/database/models"
)

// AddTag locks the document row FOR UPDATE, scoped to id + tenant, so concurrent
// tag changes serialize on that row rather than losing each other's updates.
func TestAddTag_LocksDocumentForUpdate(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.AddTag(tenantCtx(), "d-1", "guide")

	// The lock select is the first query; the mock returns no row, so the tag
	// write never runs and this is the captured query.
	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "documents"`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`FOR UPDATE`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

// RemoveTag takes the same tenant-scoped FOR UPDATE lock as AddTag.
func TestRemoveTag_LocksDocumentForUpdate(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.RemoveTag(tenantCtx(), "d-1", "guide")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "documents"`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`FOR UPDATE`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

// AddTag on a DRAFT document, driven past the lock: after locking the row it
// writes the new tag set (and bumps updated_at), scoped to id + tenant, and —
// because the document is unpublished — enqueues no reprojection.
func TestAddTag_Draft_WritesTagsNoReproject(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil)) // lock select — draft (no published_version_id)
	cfg.PushRowData(docRow(nil)) // setTags RETURNING
	_, _ = st.AddTag(tenantCtx(), "d-1", "guide")

	// The tags write is the second query; the tag set is bound as a pq array.
	set := queryAt(t, capture, 1)
	wantSQL(t, set,
		`UPDATE "documents" SET`,
		`"tags" = ?`,
		`"updated_at" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
	)
	wantArg(t, set, `{"guide"}`)

	// A draft writes Postgres only: exactly the lock and the tags update, no
	// outbox insert.
	if len(capture.Queries) != 2 {
		t.Errorf("draft tag change issued %d queries, want 2 (lock + update, no enqueue): %+v", len(capture.Queries), capture.Queries)
	}
}

// AddTag on a PUBLISHED document reprojects in the same transaction: it writes
// the tags, then — seeing a published pointer — loads the published version and
// enqueues an index job, WITHOUT moving the pointer.
func TestAddTag_Published_WritesTagsAndEnqueuesReprojection(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow("v-1"))  // lock select — published
	cfg.PushRowData(docRow("v-1"))  // setTags RETURNING (still published)
	cfg.PushRowData(versionRow())   // Versions.Get(publishedVersionID)
	cfg.PushRowData(jobRow())       // reprojection Jobs.Enqueue RETURNING
	_, _ = st.AddTag(tenantCtx(), "d-1", "guide")

	// The tags update does NOT touch published_version_id — a tag change is
	// metadata, not a publish.
	set := queryAt(t, capture, 1)
	wantSQL(t, set, `UPDATE "documents" SET`, `"tags" = ?`)
	// The SET clause assigns tags, not the pointer (it appears only in RETURNING).
	notSQL(t, set, `"published_version_id" =`)

	// A reprojection index job is enqueued.
	enqueue := queryAt(t, capture, 3)
	wantSQL(t, enqueue, `INSERT INTO "jobs"`, `ON CONFLICT`)
	wantArg(t, enqueue, models.JobIndex)
}
