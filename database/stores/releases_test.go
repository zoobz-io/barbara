//go:build testing

package stores

import (
	"context"
	"errors"
	"testing"

	"github.com/zoobz-io/grub/mockdb"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// releaseRow is a releases row.
func releaseRow(id string, number int) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "app_id", "tenant_id", "number", "created_by"},
		Rows:    [][]any{{id, testApp, testTenant, int64(number), testUser}},
	}
}

// releaseEntryRow is a release_entries row.
func releaseEntryRow(key, docID, versionID string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "release_id", "key", "document_id", "version_id"},
		Rows:    [][]any{{"e-1", "r-old", key, docID, versionID}},
	}
}

// appRowWithRelease is an apps row already pointing at a current release, so a
// cut computes its projection diff against that release.
func appRowWithRelease(releaseID string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "tenant_id", "name", "current_release_id"},
		Rows:    [][]any{{testApp, testTenant, "site", releaseID}},
	}
}

// entryRowFor is a release_entries row for a specific document.
func entryRowFor(key, docID, versionID string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "release_id", "key", "document_id", "version_id"},
		Rows:    [][]any{{"e-x", "r-prev", key, docID, versionID}},
	}
}

// noRows is an empty result set for a query that should return nothing.
func noRows() *mockdb.RowData { return &mockdb.RowData{Columns: []string{"id"}} }

// Cut over an empty tree: lock the app, number the release count+1, write it, and
// move the pointer — no entries.
func TestReleases_Cut_MovesPointerWithMonotonicNumber(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(noRows())             // snapshotHeads: no live documents
	cfg.PushRowData(appRow())             // app lock (FOR UPDATE)
	cfg.PushRowData(countRow(2))          // existing releases → next number 3
	cfg.PushRowData(releaseRow("r-1", 3)) // INSERT release RETURNING
	cfg.PushRowData(appRow())             // app pointer UPDATE RETURNING

	if _, err := st.Releases.Cut(tenantCtx(), testApp); err != nil {
		t.Fatalf("cut: %v", err)
	}

	lock := queryAt(t, capture, 1)
	wantSQL(t, lock, `FROM "apps"`, `"id" = ?`, `"tenant_id" = ?`, `FOR UPDATE`)

	ins := queryAt(t, capture, 3)
	wantSQL(t, ins, `INSERT INTO "releases"`, `"app_id"`, `"number"`, `"created_by"`, `RETURNING`)
	wantArg(t, ins, 3) // count(2) + 1
	wantArg(t, ins, testUser)

	upd := queryAt(t, capture, 4)
	wantSQL(t, upd, `UPDATE "apps" SET`, `"current_release_id" = ?`, `"id" = ?`)
}

// A full-tree cut snapshots each live document's head version into an entry.
func TestReleases_Cut_SnapshotsHeadVersions(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil))                           // snapshotHeads: one live document (d-1, a.md)
	cfg.PushRowData(versionRow())                          // its head version (v-1)
	cfg.PushRowData(appRow())                              // app lock
	cfg.PushRowData(countRow(0))                           // first release → number 1
	cfg.PushRowData(releaseRow("r-1", 1))                  // INSERT release
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // INSERT entry
	cfg.PushRowData(appRow())                              // pointer UPDATE
	// projection (prev release nil → the one entry is an add): load doc, load
	// version, enqueue one index job.
	cfg.PushRowData(docRow(nil))
	cfg.PushRowData(versionRow())
	cfg.PushRowData(jobRow())

	if _, err := st.Releases.Cut(tenantCtx(), testApp); err != nil {
		t.Fatalf("cut: %v", err)
	}

	docs := queryAt(t, capture, 0)
	wantSQL(t, docs, `FROM "documents"`, `"app_id" = ?`, `"deleted_at" IS NULL`)

	entry := queryAt(t, capture, 5)
	wantSQL(t, entry, `INSERT INTO "release_entries"`, `"key"`, `"document_id"`, `"version_id"`, `RETURNING`)
	wantArg(t, entry, "a.md")
	wantArg(t, entry, "d-1")
	wantArg(t, entry, "v-1")
}

// A cut against a previous release projects the diff: an index job for the
// added/changed path and a delete job for the removed one.
func TestReleases_Cut_ProjectsDiff(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil))                           // snapshotHeads: live doc d-1 (a.md)
	cfg.PushRowData(versionRow())                          // its head v-1
	cfg.PushRowData(appRowWithRelease("r-prev"))           // app lock — has a current release
	cfg.PushRowData(countRow(1))                           // → number 2
	cfg.PushRowData(releaseRow("r-new", 2))                // release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entry insert
	cfg.PushRowData(appRow())                              // pointer update
	cfg.PushRowData(entryRowFor("old.md", "d-2", "v-9"))   // prev entries: only d-2 (now gone)
	cfg.PushRowData(docRow(nil))                           // upsert d-1: doc load
	cfg.PushRowData(versionRow())                          // upsert d-1: version load
	cfg.PushRowData(jobRow())                              // index job for d-1
	cfg.PushRowData(jobRow())                              // delete job for d-2

	if _, err := st.Releases.Cut(tenantCtx(), testApp); err != nil {
		t.Fatalf("cut: %v", err)
	}

	// The added/changed path d-1 gets an index job; the removed d-2 a delete job.
	idx := queryAt(t, capture, 10)
	wantSQL(t, idx, `INSERT INTO "jobs"`, `"operation"`, `"document_id"`)
	wantArg(t, idx, models.JobIndex)
	wantArg(t, idx, "d-1")

	del := queryAt(t, capture, 11)
	wantSQL(t, del, `INSERT INTO "jobs"`)
	wantArg(t, del, models.JobDelete)
	wantArg(t, del, "d-2")
}

// An unchanged path (same document at the same version and key) enqueues no
// projection job.
func TestReleases_Cut_UnchangedPathSkipped(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil))                           // snapshotHeads: d-1 (a.md)
	cfg.PushRowData(versionRow())                          // head v-1
	cfg.PushRowData(appRowWithRelease("r-prev"))           // app lock with a current release
	cfg.PushRowData(countRow(1))                           // number 2
	cfg.PushRowData(releaseRow("r-new", 2))                // release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entry insert
	cfg.PushRowData(appRow())                              // pointer update
	cfg.PushRowData(entryRowFor("a.md", "d-1", "v-1"))     // prev entries: SAME d-1@v-1@a.md

	if _, err := st.Releases.Cut(tenantCtx(), testApp); err != nil {
		t.Fatalf("cut: %v", err)
	}
	// Nothing changed between releases, so no index or delete job is written.
	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "jobs"`)
	}
}

// If the projection can't be built (a document behind an entry fails to load),
// the whole cut fails and rolls back rather than committing a half-projected
// release.
func TestReleases_Cut_ProjectionLoadFailureRollsBack(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil))                           // snapshotHeads: d-1
	cfg.PushRowData(versionRow())                          // head v-1
	cfg.PushRowData(appRow())                              // app lock (no prev release)
	cfg.PushRowData(countRow(0))                           // number 1
	cfg.PushRowData(releaseRow("r-1", 1))                  // release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entry insert
	cfg.PushRowData(appRow())                              // pointer update
	cfg.PushQueryErr(errors.New("doc load boom"))          // projection: doc load fails

	if _, err := st.Releases.Cut(tenantCtx(), testApp); err == nil {
		t.Fatal("expected the cut to fail when the projection load errors")
	}
}

// List is app- and tenant-scoped, newest number first, paginated.
func TestReleases_List_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Releases.List(tenantCtx(), testApp, 10, 5)

	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "releases"`, `"app_id" = ?`, `"tenant_id" = ?`,
		`ORDER BY "number" DESC`, `LIMIT 10`, `OFFSET 5`)
}

// Get loads the release scoped to app+tenant, then its entries by key.
func TestReleases_Get_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(releaseRow("r-1", 1))                  // release select
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entries query

	if _, _, err := st.Releases.Get(tenantCtx(), testApp, "r-1"); err != nil {
		t.Fatalf("get: %v", err)
	}

	rel := queryAt(t, capture, 0)
	wantSQL(t, rel, `FROM "releases"`, `"id" = ?`, `"app_id" = ?`, `"tenant_id" = ?`)
	entries := queryAt(t, capture, 1)
	wantSQL(t, entries, `FROM "release_entries"`, `"release_id" = ?`, `ORDER BY "key" ASC`)
}

// Rollback loads an old release's entries and cuts a NEW forward release copying
// them — the number advances, the entry is copied.
func TestReleases_Rollback_CopiesEntriesForward(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(releaseRow("r-old", 1))                // old release select
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // old entries
	cfg.PushRowData(appRow())                              // app lock
	cfg.PushRowData(countRow(2))                           // → new number 3
	cfg.PushRowData(releaseRow("r-new", 3))                // new release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // copied entry insert
	cfg.PushRowData(appRow())                              // pointer update
	// projection (prev release nil in the mock → the copied entry is an add).
	cfg.PushRowData(docRow(nil))
	cfg.PushRowData(versionRow())
	cfg.PushRowData(jobRow())

	if _, err := st.Releases.Rollback(tenantCtx(), testApp, "r-old"); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	ins := queryAt(t, capture, 4)
	wantSQL(t, ins, `INSERT INTO "releases"`, `RETURNING`)
	wantArg(t, ins, 3) // a new forward number, never backward

	entry := queryAt(t, capture, 5)
	wantSQL(t, entry, `INSERT INTO "release_entries"`)
	wantArg(t, entry, "d-1") // the old entry, copied
	wantArg(t, entry, "v-1")
}

// CurrentEntries loads the app's current release entries; an app with no current
// release returns none without touching the entries table.
func TestReleases_CurrentEntries_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRowWithRelease("r-1"))          // apps.Get: has a current release
	cfg.PushRowData(releaseRow("r-1", 1))              // getTx: release select
	cfg.PushRowData(entryRowFor("a.md", "d-1", "v-1")) // getTx: entries

	entries, err := st.Releases.CurrentEntries(tenantCtx(), testApp)
	if err != nil || len(entries) != 1 || entries[0].DocumentID != "d-1" {
		t.Fatalf("current entries = %+v, %v; want the one live entry", entries, err)
	}
	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "release_entries"`, `"release_id" = ?`)
}

// No current release → no entries, and the entries table is never queried.
func TestReleases_CurrentEntries_NoRelease(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow()) // apps.Get: current_release_id nil

	entries, err := st.Releases.CurrentEntries(tenantCtx(), testApp)
	if err != nil || entries != nil {
		t.Fatalf("current entries = %+v, %v; want none", entries, err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `FROM "release_entries"`)
	}
}

// CurrentEntryFor returns a document's entry in the current release.
func TestReleases_CurrentEntryFor_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRowWithRelease("r-1"))
	cfg.PushRowData(entryRowFor("a.md", "d-1", "v-1"))

	entry, err := st.Releases.CurrentEntryFor(tenantCtx(), testApp, "d-1")
	if err != nil || entry == nil || entry.VersionID != "v-1" {
		t.Fatalf("current entry = %+v, %v; want d-1 at v-1", entry, err)
	}
	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "release_entries"`, `"release_id" = ?`, `"document_id" = ?`)
}

// Contains counts a document's entries in a release.
func TestReleases_Contains_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(1))
	ok, err := st.Releases.Contains(tenantCtx(), "r-1", "d-1")
	if err != nil || !ok {
		t.Fatalf("contains = %v, %v; want true, nil", ok, err)
	}
	q := lastQuery(t, capture)
	wantSQL(t, q, `SELECT COUNT(*) FROM "release_entries"`, `"release_id" = ?`, `"document_id" = ?`)
}

// Cutting requires an acting user — the release records who cut it.
func TestReleases_Cut_RequiresUser(t *testing.T) {
	st, _ := newQueryTest(t)
	// A tenant but no user.
	ctx := auth.WithPrincipal(context.Background(), auth.NewPrincipal("", testTenant, "", nil, nil))
	if _, err := st.Releases.Cut(ctx, testApp); !errors.Is(err, auth.ErrNoUser) {
		t.Errorf("cut without user = %v, want ErrNoUser", err)
	}
	if _, err := st.Releases.Cut(context.Background(), testApp); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("cut without tenant = %v, want ErrNoTenant", err)
	}
}

// Cut emits Release.Cut carrying the new number.
func TestReleases_Cut_EmitsCut(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(noRows())
	cfg.PushRowData(appRow())
	cfg.PushRowData(countRow(0))
	cfg.PushRowData(releaseRow("r-1", 1))
	cfg.PushRowData(appRow())

	var got events.ReleaseCutEvent
	fired := false
	l := events.Release.Cut.Listen(func(_ context.Context, e events.ReleaseCutEvent) { got, fired = e, true })
	defer l.Close()

	if _, err := st.Releases.Cut(tenantCtx(), testApp); err != nil {
		t.Fatalf("cut: %v", err)
	}
	if !fired || got.ReleaseID != "r-1" || got.AppID != testApp || got.Number != 1 {
		t.Errorf("Cut event = %+v (fired=%v)", got, fired)
	}
}
