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
		Columns: []string{"id", "app_id", "tenant_id", "number", "created_by", "kind"},
		Rows:    [][]any{{id, testApp, testTenant, int64(number), testUser, models.ReleaseKindCut}},
	}
}

// releaseEntryRow is a release_entries row.
func releaseEntryRow(key, docID, versionID string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "release_id", "key", "document_id", "version_id", "version_number"},
		Rows:    [][]any{{"e-1", "r-old", key, docID, versionID, int64(1)}},
	}
}

// changeRow is a release_changes row — enough for an INSERT ... RETURNING to scan.
func changeRow(key, docID, change string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "release_id", "key", "document_id", "change"},
		Rows:    [][]any{{"c-1", "r-new", key, docID, change}},
	}
}

// appRowWithRelease is an apps row already pointing at a current release, so a
// cut diffs against that release.
func appRowWithRelease(releaseID string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "tenant_id", "name", "current_release_id"},
		Rows:    [][]any{{testApp, testTenant, "site", releaseID}},
	}
}

// entryRowFor is a release_entries row for a specific document.
func entryRowFor(key, docID, versionID string) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "release_id", "key", "document_id", "version_id", "version_number"},
		Rows:    [][]any{{"e-x", "r-prev", key, docID, versionID, int64(1)}},
	}
}

// noRows is an empty result set for a query that should return nothing.
func noRows() *mockdb.RowData { return &mockdb.RowData{Columns: []string{"id"}} }

// Cut over an empty tree: lock the app, number the release count+1, write it
// with its kind, and move the pointer — no entries, no changes.
func TestReleases_Cut_MovesPointerWithMonotonicNumber(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(noRows())             // snapshotHeads: no live documents
	cfg.PushRowData(appRow())             // app lock (FOR UPDATE)
	cfg.PushRowData(countRow(2))          // existing releases → next number 3
	cfg.PushRowData(releaseRow("r-1", 3)) // INSERT release RETURNING
	cfg.PushRowData(appRow())             // app pointer UPDATE RETURNING

	if _, err := st.Releases.Cut(tenantCtx(), testApp, ""); err != nil {
		t.Fatalf("cut: %v", err)
	}

	lock := queryAt(t, capture, 1)
	wantSQL(t, lock, `FROM "apps"`, `"id" = ?`, `"tenant_id" = ?`, `FOR UPDATE`)

	ins := queryAt(t, capture, 3)
	wantSQL(t, ins, `INSERT INTO "releases"`, `"app_id"`, `"number"`, `"created_by"`, `"kind"`, `"entry_count"`, `RETURNING`)
	wantArg(t, ins, 3) // count(2) + 1
	wantArg(t, ins, testUser)
	wantArg(t, ins, models.ReleaseKindCut)

	upd := queryAt(t, capture, 4)
	wantSQL(t, upd, `UPDATE "apps" SET`, `"current_release_id" = ?`, `"id" = ?`)

	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "release_changes"`)
	}
}

// A full-tree cut snapshots each live document's head version — id and number —
// into an entry, and records the first release's paths as additions.
func TestReleases_Cut_SnapshotsHeadVersions(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))                       // snapshotHeads: one live document (d-1, a.md)
	cfg.PushRowData(versionRow())                          // its head version (v-1, number 1)
	cfg.PushRowData(appRow())                              // app lock
	cfg.PushRowData(countRow(0))                           // first release → number 1
	cfg.PushRowData(releaseRow("r-1", 1))                  // INSERT release
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // INSERT entry
	cfg.PushRowData(changeRow("a.md", "d-1", "added"))     // INSERT change
	cfg.PushRowData(appRow())                              // pointer UPDATE
	// projection (no previous release → the one entry is an add): load doc, load
	// version, enqueue one index job.
	cfg.PushRowData(docRow(testApp))
	cfg.PushRowData(versionRow())
	cfg.PushRowData(jobRow())

	if _, err := st.Releases.Cut(tenantCtx(), testApp, "  "); err != nil {
		t.Fatalf("cut: %v", err)
	}

	docs := queryAt(t, capture, 0)
	wantSQL(t, docs, `FROM "documents"`, `"app_id" = ?`, `"deleted_at" IS NULL`)

	rel := queryAt(t, capture, 4)
	wantSQL(t, rel, `INSERT INTO "releases"`, `"added"`, `"label"`)
	wantArg(t, rel, 1) // entry_count 1, added 1
	wantArg(t, rel, nil)

	entry := queryAt(t, capture, 5)
	wantSQL(t, entry, `INSERT INTO "release_entries"`, `"key"`, `"document_id"`, `"version_id"`, `"version_number"`, `RETURNING`)
	wantArg(t, entry, "a.md")
	wantArg(t, entry, "d-1")
	wantArg(t, entry, "v-1")
	wantArg(t, entry, 1)

	change := queryAt(t, capture, 6)
	wantSQL(t, change, `INSERT INTO "release_changes"`, `"change"`)
	wantArg(t, change, models.ChangeAdded)
	wantArg(t, change, "d-1")
}

// A cut against a previous release diffs its entry set: a change row for the
// added path and the removed one, counts on the release row, an index job for
// the added path and a delete job for the removed one.
func TestReleases_Cut_ProjectsDiff(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))                       // snapshotHeads: live doc d-1 (a.md)
	cfg.PushRowData(versionRow())                          // its head v-1
	cfg.PushRowData(appRowWithRelease("r-prev"))           // app lock — has a current release
	cfg.PushRowData(countRow(1))                           // → number 2
	cfg.PushRowData(entryRowFor("old.md", "d-2", "v-9"))   // prev entries: only d-2 (now gone)
	cfg.PushRowData(releaseRow("r-new", 2))                // release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entry insert
	cfg.PushRowData(changeRow("a.md", "d-1", "added"))     // change: d-1 added
	cfg.PushRowData(changeRow("old.md", "d-2", "removed")) // change: d-2 removed
	cfg.PushRowData(appRow())                              // pointer update
	cfg.PushRowData(docRow(testApp))                       // index d-1: doc load
	cfg.PushRowData(versionRow())                          // index d-1: version load
	cfg.PushRowData(jobRow())                              // index job for d-1
	cfg.PushRowData(jobRow())                              // delete job for d-2

	if _, err := st.Releases.Cut(tenantCtx(), testApp, "Launch"); err != nil {
		t.Fatalf("cut: %v", err)
	}

	prev := queryAt(t, capture, 4)
	wantSQL(t, prev, `FROM "release_entries"`, `"release_id" = ?`)
	wantArg(t, prev, "r-prev")

	rel := queryAt(t, capture, 5)
	wantSQL(t, rel, `INSERT INTO "releases"`, `"added"`, `"removed"`)
	wantArg(t, rel, "Launch")

	added := queryAt(t, capture, 7)
	wantSQL(t, added, `INSERT INTO "release_changes"`)
	wantArg(t, added, models.ChangeAdded)
	wantArg(t, added, "d-1")
	removed := queryAt(t, capture, 8)
	wantSQL(t, removed, `INSERT INTO "release_changes"`)
	wantArg(t, removed, models.ChangeRemoved)
	wantArg(t, removed, "d-2")
	wantArg(t, removed, "v-9") // the version that went away

	// The added path d-1 gets an index job; the removed d-2 a delete job.
	idx := queryAt(t, capture, 12)
	wantSQL(t, idx, `INSERT INTO "jobs"`, `"operation"`, `"document_id"`)
	wantArg(t, idx, models.JobIndex)
	wantArg(t, idx, "d-1")

	del := queryAt(t, capture, 13)
	wantSQL(t, del, `INSERT INTO "jobs"`)
	wantArg(t, del, models.JobDelete)
	wantArg(t, del, "d-2")
}

// An unchanged path (same document at the same version and key) writes no
// change row and enqueues no projection job.
func TestReleases_Cut_UnchangedPathSkipped(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))                       // snapshotHeads: d-1 (a.md)
	cfg.PushRowData(versionRow())                          // head v-1
	cfg.PushRowData(appRowWithRelease("r-prev"))           // app lock with a current release
	cfg.PushRowData(countRow(1))                           // number 2
	cfg.PushRowData(entryRowFor("a.md", "d-1", "v-1"))     // prev entries: SAME d-1@v-1@a.md
	cfg.PushRowData(releaseRow("r-new", 2))                // release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entry insert
	cfg.PushRowData(appRow())                              // pointer update

	if _, err := st.Releases.Cut(tenantCtx(), testApp, ""); err != nil {
		t.Fatalf("cut: %v", err)
	}
	// Nothing changed between releases, so no change row, index, or delete job
	// is written.
	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "jobs"`)
		notSQL(t, q, `INSERT INTO "release_changes"`)
	}
}

// If the projection can't be built (a document behind an entry fails to load),
// the whole cut fails and rolls back rather than committing a half-projected
// release.
func TestReleases_Cut_ProjectionLoadFailureRollsBack(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))                       // snapshotHeads: d-1
	cfg.PushRowData(versionRow())                          // head v-1
	cfg.PushRowData(appRow())                              // app lock (no prev release)
	cfg.PushRowData(countRow(0))                           // number 1
	cfg.PushRowData(releaseRow("r-1", 1))                  // release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // entry insert
	cfg.PushRowData(changeRow("a.md", "d-1", "added"))     // change insert
	cfg.PushRowData(appRow())                              // pointer update
	cfg.PushQueryErr(errors.New("doc load boom"))          // projection: doc load fails

	if _, err := st.Releases.Cut(tenantCtx(), testApp, ""); err == nil {
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

// Total counts the app's releases, tenant-scoped.
func TestReleases_Total_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(7))

	n, err := st.Releases.Total(tenantCtx(), testApp)
	if err != nil || n != 7 {
		t.Fatalf("total = %d, %v; want 7", n, err)
	}
	q := lastQuery(t, capture)
	wantSQL(t, q, `SELECT COUNT(*) FROM "releases"`, `"app_id" = ?`, `"tenant_id" = ?`)
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

// Changes loads the release scoped to app+tenant, then its change rows by key.
func TestReleases_Changes_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(releaseRow("r-1", 1))              // release select
	cfg.PushRowData(changeRow("a.md", "d-1", "added")) // changes query

	_, changes, err := st.Releases.Changes(tenantCtx(), testApp, "r-1")
	if err != nil || len(changes) != 1 || changes[0].Change != models.ChangeAdded {
		t.Fatalf("changes = %+v, %v; want the one added row", changes, err)
	}

	rel := queryAt(t, capture, 0)
	wantSQL(t, rel, `FROM "releases"`, `"id" = ?`, `"app_id" = ?`, `"tenant_id" = ?`)
	q := queryAt(t, capture, 1)
	wantSQL(t, q, `FROM "release_changes"`, `"release_id" = ?`, `ORDER BY "key" ASC`)
}

// A release that is not the app's is not found, and its changes never load.
func TestReleases_Changes_NotFound(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(noRows())

	if _, _, err := st.Releases.Changes(tenantCtx(), testApp, "r-x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("changes of a foreign release = %v, want ErrNotFound", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `FROM "release_changes"`)
	}
}

// Rollback loads an old release's entries and cuts a NEW forward release copying
// them — the number advances, the entry is copied, the kind and source are
// recorded.
func TestReleases_Rollback_CopiesEntriesForward(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(releaseRow("r-old", 1))                // old release select
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // old entries
	cfg.PushRowData(appRow())                              // app lock (no current release in the mock)
	cfg.PushRowData(countRow(2))                           // → new number 3
	cfg.PushRowData(releaseRow("r-new", 3))                // new release insert
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // copied entry insert
	cfg.PushRowData(changeRow("a.md", "d-1", "added"))     // the copied entry is an add against nothing
	cfg.PushRowData(appRow())                              // pointer update
	cfg.PushRowData(docRow(testApp))                       // projection: doc load
	cfg.PushRowData(versionRow())                          // projection: version load
	cfg.PushRowData(jobRow())                              // index job

	if _, err := st.Releases.Rollback(tenantCtx(), testApp, "r-old", "back to 1"); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	ins := queryAt(t, capture, 4)
	wantSQL(t, ins, `INSERT INTO "releases"`, `"source_release_id"`, `RETURNING`)
	wantArg(t, ins, 3) // a new forward number, never backward
	wantArg(t, ins, models.ReleaseKindRollback)
	wantArg(t, ins, "r-old")
	wantArg(t, ins, "back to 1")

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
	if _, err := st.Releases.Cut(ctx, testApp, ""); !errors.Is(err, auth.ErrNoUser) {
		t.Errorf("cut without user = %v, want ErrNoUser", err)
	}
	if _, err := st.Releases.Cut(context.Background(), testApp, ""); !errors.Is(err, auth.ErrNoTenant) {
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

	if _, err := st.Releases.Cut(tenantCtx(), testApp, ""); err != nil {
		t.Fatalf("cut: %v", err)
	}
	if !fired || got.ReleaseID != "r-1" || got.AppID != testApp || got.Number != 1 {
		t.Errorf("Cut event = %+v (fired=%v)", got, fired)
	}
}

// --- the diff ---

func entry(key, doc, version string, number int) *models.ReleaseEntry {
	return &models.ReleaseEntry{Key: key, DocumentID: doc, VersionID: version, VersionNumber: number}
}

func spec(key, doc, version string, number int) ReleaseEntrySpec {
	return ReleaseEntrySpec{Key: key, DocumentID: doc, VersionID: version, VersionNumber: number}
}

// diffEntries classifies every document by id: added, changed at the same
// key, moved to a new key, removed, or (no row) unchanged — new entries first
// in their order, then removals in the previous release's order.
func TestDiffEntries_Classifies(t *testing.T) {
	prev := []*models.ReleaseEntry{
		entry("a.md", "d-a", "v-a1", 1), // unchanged
		entry("b.md", "d-b", "v-b1", 1), // changed → v-b2
		entry("c.md", "d-c", "v-c1", 1), // moved → docs/c.md, same version
		entry("d.md", "d-d", "v-d1", 1), // moved AND edited
		entry("e.md", "d-e", "v-e1", 1), // removed
	}
	next := []ReleaseEntrySpec{
		spec("a.md", "d-a", "v-a1", 1),
		spec("b.md", "d-b", "v-b2", 2),
		spec("docs/c.md", "d-c", "v-c1", 1),
		spec("docs/d.md", "d-d", "v-d2", 2),
		spec("f.md", "d-f", "v-f1", 1), // added
	}

	d := diffEntries(prev, next)

	if d.added != 1 || d.changed != 1 || d.moved != 2 || d.removed != 1 {
		t.Fatalf("counts = added %d changed %d moved %d removed %d; want 1/1/2/1", d.added, d.changed, d.moved, d.removed)
	}
	if len(d.changes) != 5 {
		t.Fatalf("changes = %d, want 5 (unchanged a.md has no row)", len(d.changes))
	}
	want := []struct {
		key, doc, change string
		prevKey          string
		prevNumber       int // 0 = absent
		number           int // 0 = absent
	}{
		{"b.md", "d-b", models.ChangeChanged, "", 1, 2},
		{"docs/c.md", "d-c", models.ChangeMoved, "c.md", 1, 1},
		{"docs/d.md", "d-d", models.ChangeMoved, "d.md", 1, 2},
		{"f.md", "d-f", models.ChangeAdded, "", 0, 1},
		{"e.md", "d-e", models.ChangeRemoved, "", 1, 0},
	}
	for i, w := range want {
		got := d.changes[i]
		if got.Key != w.key || got.DocumentID != w.doc || got.Change != w.change {
			t.Errorf("change[%d] = %s %s %s, want %s %s %s", i, got.Key, got.DocumentID, got.Change, w.key, w.doc, w.change)
		}
		if (got.PrevKey == nil) != (w.prevKey == "") || (got.PrevKey != nil && *got.PrevKey != w.prevKey) {
			t.Errorf("change[%d] prev key = %v, want %q", i, got.PrevKey, w.prevKey)
		}
		if (got.PrevVersionNumber == nil) != (w.prevNumber == 0) || (got.PrevVersionNumber != nil && *got.PrevVersionNumber != w.prevNumber) {
			t.Errorf("change[%d] prev number = %v, want %d", i, got.PrevVersionNumber, w.prevNumber)
		}
		if (got.VersionNumber == nil) != (w.number == 0) || (got.VersionNumber != nil && *got.VersionNumber != w.number) {
			t.Errorf("change[%d] number = %v, want %d", i, got.VersionNumber, w.number)
		}
	}
}

// With no previous release every entry is an addition; with no next entries
// every previous entry is a removal.
func TestDiffEntries_Edges(t *testing.T) {
	first := diffEntries(nil, []ReleaseEntrySpec{spec("a.md", "d-a", "v-a1", 1), spec("b.md", "d-b", "v-b1", 1)})
	if first.added != 2 || len(first.changes) != 2 || first.changes[0].PrevVersionID != nil {
		t.Errorf("first release diff = %+v, want two additions", first)
	}
	last := diffEntries([]*models.ReleaseEntry{entry("a.md", "d-a", "v-a1", 1)}, nil)
	if last.removed != 1 || len(last.changes) != 1 || last.changes[0].VersionID != nil {
		t.Errorf("unpublish-all diff = %+v, want one removal", last)
	}
	if none := diffEntries(nil, nil); len(none.changes) != 0 {
		t.Errorf("empty diff = %+v, want none", none)
	}
}

// A key freed by one document and retaken by another in one cut is a removal
// and an addition, not a change — the diff is keyed by document.
func TestDiffEntries_KeyRetaken(t *testing.T) {
	d := diffEntries(
		[]*models.ReleaseEntry{entry("a.md", "d-old", "v-1", 1)},
		[]ReleaseEntrySpec{spec("a.md", "d-new", "v-9", 1)},
	)
	if d.added != 1 || d.removed != 1 || d.changed != 0 || len(d.changes) != 2 {
		t.Fatalf("retaken key diff = %+v, want one addition and one removal", d)
	}
	if d.changes[0].DocumentID != "d-new" || d.changes[1].DocumentID != "d-old" {
		t.Errorf("retaken key rows = %s, %s; want d-new then d-old", d.changes[0].DocumentID, d.changes[1].DocumentID)
	}
}
