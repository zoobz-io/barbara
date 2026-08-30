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
	cfg.PushRowData(docRow(testApp)) // lock select — a draft
	cfg.PushRowData(docRow(testApp)) // setTags RETURNING
	cfg.PushRowData(appRow())        // CurrentEntryFor: app has no current release
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

	// A draft writes Postgres only: the lock, the tags update, and the
	// current-release check that finds nothing — no outbox insert.
	if len(capture.Queries) != 3 {
		t.Errorf("draft tag change issued %d queries, want 3 (lock + update + release check, no enqueue): %+v", len(capture.Queries), capture.Queries)
	}
}

// AddTag on a LIVE document (carried by the current release) reprojects in the
// same transaction: it writes the tags, then — finding the document in the
// current release — loads the release-served version and enqueues an index job,
// without cutting a new release.
func TestAddTag_Live_WritesTagsAndEnqueuesReprojection(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))                  // lock select — placed
	cfg.PushRowData(docRow(testApp))                  // setTags RETURNING
	cfg.PushRowData(appRowWithRelease("r-1"))         // CurrentEntryFor: apps.Get (has a release)
	cfg.PushRowData(entryRowFor("a.md", "d-1", "v-1")) // ...which carries d-1 at v-1
	cfg.PushRowData(versionRow())                     // enqueueReprojection: Versions.Get(v-1)
	cfg.PushRowData(jobRow())                         // Jobs.Enqueue RETURNING
	_, _ = st.AddTag(tenantCtx(), "d-1", "guide")

	set := queryAt(t, capture, 1)
	wantSQL(t, set, `UPDATE "documents" SET`, `"tags" = ?`)

	// A reprojection index job is enqueued off the release entry.
	enqueue := queryAt(t, capture, 5)
	wantSQL(t, enqueue, `INSERT INTO "jobs"`, `ON CONFLICT`)
	wantArg(t, enqueue, models.JobIndex)
}
