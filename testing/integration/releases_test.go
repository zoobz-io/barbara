//go:build testing

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/zoobz-io/barbara/database/models"
)

// entryFor returns the entry at the given key, or nil.
func entryFor(entries []*models.ReleaseEntry, key string) *models.ReleaseEntry {
	for _, e := range entries {
		if e.Key == key {
			return e
		}
	}
	return nil
}

// TestReleases_Lifecycle drives the release primitive against real Postgres: a
// full-tree cut snapshots head versions, the pointer moves, numbers stay
// monotonic, get returns entries, and rollback cuts a new forward release.
func TestReleases_Lifecycle(t *testing.T) {
	st, db, cleanup := newDocStores(t)
	t.Cleanup(cleanup)
	ctx := tenantCtx(testTenant)

	app, err := st.Apps.Create(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	// Two documents, each with a head version.
	a, _ := st.Documents.Create(ctx, app.ID, nil, "a.md")
	v1a, err := st.Versions.Save(ctx, a.ID, "a v1", 0)
	if err != nil {
		t.Fatalf("save a v1: %v", err)
	}
	b, _ := st.Documents.Create(ctx, app.ID, nil, "b.md")
	if _, err := st.Versions.Save(ctx, b.ID, "b v1", 0); err != nil {
		t.Fatalf("save b v1: %v", err)
	}
	// A third document with no version — it must not appear in the release.
	if _, err := st.Documents.Create(ctx, app.ID, nil, "draft.md"); err != nil {
		t.Fatalf("create draft: %v", err)
	}

	// Cut release 1: snapshots the two documents that have content.
	r1, err := st.Releases.Cut(ctx, app.ID, "")
	if err != nil {
		t.Fatalf("cut r1: %v", err)
	}
	if r1.Number != 1 {
		t.Errorf("first release number = %d, want 1", r1.Number)
	}
	if got := appPointer(t, db, app.ID); got != r1.ID {
		t.Errorf("pointer = %s, want r1 %s", got, r1.ID)
	}
	_, entries, err := st.Releases.Get(ctx, app.ID, r1.ID)
	if err != nil {
		t.Fatalf("get r1: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("r1 entries = %d, want 2 (the drafted doc excluded)", len(entries))
	}
	if entries[0].Key != "a.md" || entries[0].VersionID != v1a.ID {
		t.Errorf("first entry = %+v, want a.md at v1", entries[0])
	}

	// Edit a, cut release 2: number advances, the pointer moves.
	v2a, _ := st.Versions.Save(ctx, a.ID, "a v2", 1)
	r2, err := st.Releases.Cut(ctx, app.ID, "")
	if err != nil {
		t.Fatalf("cut r2: %v", err)
	}
	if r2.Number != 2 {
		t.Errorf("second release number = %d, want 2", r2.Number)
	}
	_, r2entries, _ := st.Releases.Get(ctx, app.ID, r2.ID)
	if aEntry := entryFor(r2entries, "a.md"); aEntry == nil || aEntry.VersionID != v2a.ID {
		t.Errorf("r2 should carry a.md at v2: %+v", r2entries)
	}

	// Rollback to r1: a NEW release (number 3) copying r1's entries; pointer never
	// moves backward.
	r3, err := st.Releases.Rollback(ctx, app.ID, r1.ID, "")
	if err != nil {
		t.Fatalf("rollback to r1: %v", err)
	}
	if r3.Number != 3 {
		t.Errorf("rollback number = %d, want 3 (forward)", r3.Number)
	}
	if got := appPointer(t, db, app.ID); got != r3.ID {
		t.Errorf("pointer after rollback = %s, want r3 %s", got, r3.ID)
	}
	_, r3entries, _ := st.Releases.Get(ctx, app.ID, r3.ID)
	if aEntry := entryFor(r3entries, "a.md"); aEntry == nil || aEntry.VersionID != v1a.ID {
		t.Errorf("rollback should restore a.md at v1: %+v", r3entries)
	}

	// List returns all three, newest first.
	list, err := st.Releases.List(ctx, app.ID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 || list[0].Number != 3 || list[2].Number != 1 {
		t.Errorf("list = %+v, want numbers 3,2,1", list)
	}
}

func appPointer(t *testing.T, db *sqlx.DB, appID string) string {
	t.Helper()
	var ptr *string
	if err := db.QueryRowx("SELECT current_release_id FROM apps WHERE id=$1", appID).Scan(&ptr); err != nil {
		t.Fatalf("read pointer: %v", err)
	}
	if ptr == nil {
		return ""
	}
	return *ptr
}

// changeFor returns the change row for the given document, or nil.
func changeFor(changes []*models.ReleaseChange, docID string) *models.ReleaseChange {
	for _, c := range changes {
		if c.DocumentID == docID {
			return c
		}
	}
	return nil
}

// counts is the at-a-glance shape of a release row.
type counts struct{ entries, added, changed, removed, moved int }

func countsOf(r *models.Release) counts {
	return counts{r.EntryCount, r.Added, r.Changed, r.Removed, r.Moved}
}

// TestReleases_MetadataAndChanges drives the metadata a cut records against
// real Postgres: kind and label on the row, version numbers on entries, and the
// diff against the previous release as counts and change rows — an addition,
// an edit, a move, and (through a rollback) a removal.
func TestReleases_MetadataAndChanges(t *testing.T) {
	st, _, cleanup := newDocStores(t)
	t.Cleanup(cleanup)
	ctx := tenantCtx(testTenant)

	app, err := st.Apps.Create(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	guides, err := st.Collections.Create(ctx, app.ID, nil, "guides")
	if err != nil {
		t.Fatalf("create guides: %v", err)
	}
	a, _ := st.Documents.Create(ctx, app.ID, nil, "a.md")
	v1a, _ := st.Versions.Save(ctx, a.ID, "a v1", 0)
	b, _ := st.Documents.Create(ctx, app.ID, nil, "b.md")
	v1b, _ := st.Versions.Save(ctx, b.ID, "b v1", 0)

	// r1: everything is an addition; the label and kind land on the row.
	r1, err := st.Releases.Cut(ctx, app.ID, "  first  ")
	if err != nil {
		t.Fatalf("cut r1: %v", err)
	}
	if r1.Kind != models.ReleaseKindCut || r1.Label == nil || *r1.Label != "first" || r1.SourceReleaseID != nil {
		t.Errorf("r1 = kind %q label %v source %v; want cut, \"first\" (trimmed), no source", r1.Kind, r1.Label, r1.SourceReleaseID)
	}
	if got := countsOf(r1); got != (counts{2, 2, 0, 0, 0}) {
		t.Errorf("r1 counts = %+v, want 2 entries, 2 added", got)
	}
	_, r1entries, _ := st.Releases.Get(ctx, app.ID, r1.ID)
	if e := entryFor(r1entries, "a.md"); e == nil || e.VersionNumber != 1 {
		t.Errorf("r1 a.md entry = %+v, want version number 1", e)
	}
	_, r1changes, err := st.Releases.Changes(ctx, app.ID, r1.ID)
	if err != nil {
		t.Fatalf("changes r1: %v", err)
	}
	if len(r1changes) != 2 || r1changes[0].Change != models.ChangeAdded || r1changes[0].PrevVersionID != nil ||
		r1changes[0].VersionID == nil || *r1changes[0].VersionID != v1a.ID || *r1changes[0].VersionNumber != 1 {
		t.Errorf("r1 changes = %+v, want two additions with the new side only", r1changes)
	}

	// Edit a, move b into guides, add c; r2 is one change, one move, one addition.
	v2a, _ := st.Versions.Save(ctx, a.ID, "a v2", 1)
	if _, err := st.Documents.Move(ctx, app.ID, b.ID, &guides.ID, "b.md"); err != nil {
		t.Fatalf("move b: %v", err)
	}
	c, _ := st.Documents.Create(ctx, app.ID, nil, "c.md")
	v1c, _ := st.Versions.Save(ctx, c.ID, "c v1", 0)
	r2, err := st.Releases.Cut(ctx, app.ID, "")
	if err != nil {
		t.Fatalf("cut r2: %v", err)
	}
	if r2.Label != nil {
		t.Errorf("r2 label = %q, want none", *r2.Label)
	}
	if got := countsOf(r2); got != (counts{3, 1, 1, 0, 1}) {
		t.Errorf("r2 counts = %+v, want 3 entries, 1 added, 1 changed, 1 moved", got)
	}
	_, r2changes, _ := st.Releases.Changes(ctx, app.ID, r2.ID)
	if ch := changeFor(r2changes, a.ID); ch == nil || ch.Change != models.ChangeChanged || ch.Key != "a.md" ||
		*ch.PrevVersionID != v1a.ID || *ch.PrevVersionNumber != 1 || *ch.VersionID != v2a.ID || *ch.VersionNumber != 2 {
		t.Errorf("r2 change for a = %+v, want changed v1 → v2", ch)
	}
	if ch := changeFor(r2changes, b.ID); ch == nil || ch.Change != models.ChangeMoved || ch.Key != "guides/b.md" ||
		ch.PrevKey == nil || *ch.PrevKey != "b.md" || *ch.PrevVersionID != v1b.ID || *ch.VersionID != v1b.ID {
		t.Errorf("r2 change for b = %+v, want moved b.md → guides/b.md at the same version", ch)
	}
	if ch := changeFor(r2changes, c.ID); ch == nil || ch.Change != models.ChangeAdded || *ch.VersionID != v1c.ID {
		t.Errorf("r2 change for c = %+v, want added", ch)
	}
	if n, _ := st.Releases.Total(ctx, app.ID); n != 2 {
		t.Errorf("total = %d, want 2", n)
	}

	// Roll back to r1: a rollback kind sourced from r1, whose diff against r2 is
	// a revert of a, a move of b back, and the removal of c.
	r3, err := st.Releases.Rollback(ctx, app.ID, r1.ID, "back to first")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if r3.Kind != models.ReleaseKindRollback || r3.SourceReleaseID == nil || *r3.SourceReleaseID != r1.ID || *r3.Label != "back to first" {
		t.Errorf("r3 = kind %q source %v label %v; want rollback from r1 with its label", r3.Kind, r3.SourceReleaseID, r3.Label)
	}
	if got := countsOf(r3); got != (counts{2, 0, 1, 1, 1}) {
		t.Errorf("r3 counts = %+v, want 2 entries, 1 changed, 1 removed, 1 moved", got)
	}
	_, r3changes, _ := st.Releases.Changes(ctx, app.ID, r3.ID)
	if ch := changeFor(r3changes, c.ID); ch == nil || ch.Change != models.ChangeRemoved || ch.Key != "c.md" ||
		ch.VersionID != nil || ch.VersionNumber != nil || *ch.PrevVersionID != v1c.ID || *ch.PrevVersionNumber != 1 {
		t.Errorf("r3 change for c = %+v, want removed with the previous side only", ch)
	}
	if ch := changeFor(r3changes, a.ID); ch == nil || ch.Change != models.ChangeChanged || *ch.PrevVersionNumber != 2 || *ch.VersionNumber != 1 {
		t.Errorf("r3 change for a = %+v, want changed v2 → v1", ch)
	}
	if ch := changeFor(r3changes, b.ID); ch == nil || ch.Change != models.ChangeMoved || ch.Key != "b.md" || *ch.PrevKey != "guides/b.md" {
		t.Errorf("r3 change for b = %+v, want moved guides/b.md → b.md", ch)
	}
	// The list carries the counts, so the timeline reads without loading entries.
	list, _ := st.Releases.List(ctx, app.ID, 50, 0)
	if len(list) != 3 || list[0].Removed != 1 || list[1].Moved != 1 || list[2].Added != 2 {
		t.Errorf("listed counts = %+v", list)
	}
}

// TestReleases_Rebuild wipes the counts and change rows the cuts wrote and
// rebuilds them from entries; the result matches what the cuts recorded.
func TestReleases_Rebuild(t *testing.T) {
	st, db, cleanup := newDocStores(t)
	t.Cleanup(cleanup)
	ctx := tenantCtx(testTenant)

	app, _ := st.Apps.Create(ctx, uuid.NewString())
	a, _ := st.Documents.Create(ctx, app.ID, nil, "a.md")
	_, _ = st.Versions.Save(ctx, a.ID, "a v1", 0)
	b, _ := st.Documents.Create(ctx, app.ID, nil, "b.md")
	_, _ = st.Versions.Save(ctx, b.ID, "b v1", 0)
	r1, _ := st.Releases.Cut(ctx, app.ID, "")
	_, _ = st.Versions.Save(ctx, a.ID, "a v2", 1)
	r2, _ := st.Releases.Cut(ctx, app.ID, "")
	if _, err := st.Unpublish(ctx, b.ID); err != nil {
		t.Fatalf("unpublish b: %v", err)
	}
	r3, _ := st.Releases.Rollback(ctx, app.ID, r1.ID, "")

	type snapshot struct {
		counts  counts
		changes []*models.ReleaseChange
	}
	snap := func(id string) snapshot {
		r, ch, err := st.Releases.Changes(ctx, app.ID, id)
		if err != nil {
			t.Fatalf("changes %s: %v", id, err)
		}
		return snapshot{countsOf(r), ch}
	}
	before := map[string]snapshot{}
	for _, id := range []string{r1.ID, r2.ID, r3.ID} {
		before[id] = snap(id)
	}
	if before[r3.ID].counts != (counts{2, 1, 1, 0, 0}) { // b back, a reverted
		t.Fatalf("r3 counts before rebuild = %+v", before[r3.ID].counts)
	}

	// Drift: the projection is gone and the counts are zero.
	if _, err := db.Exec("DELETE FROM release_changes"); err != nil {
		t.Fatalf("clearing changes: %v", err)
	}
	if _, err := db.Exec("UPDATE releases SET entry_count = 0, added = 0, changed = 0, removed = 0, moved = 0"); err != nil {
		t.Fatalf("zeroing counts: %v", err)
	}

	n, err := st.RebuildReleases(context.Background())
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if n < 3 {
		t.Errorf("rebuilt %d releases, want at least this app's 3", n)
	}
	for id, want := range before {
		got := snap(id)
		if got.counts != want.counts {
			t.Errorf("release %s counts after rebuild = %+v, want %+v", id, got.counts, want.counts)
		}
		if len(got.changes) != len(want.changes) {
			t.Errorf("release %s has %d changes after rebuild, want %d", id, len(got.changes), len(want.changes))
			continue
		}
		for i := range want.changes {
			g, w := got.changes[i], want.changes[i]
			if g.Key != w.Key || g.DocumentID != w.DocumentID || g.Change != w.Change ||
				!equalStr(g.PrevKey, w.PrevKey) || !equalStr(g.PrevVersionID, w.PrevVersionID) || !equalStr(g.VersionID, w.VersionID) ||
				!equalInt(g.PrevVersionNumber, w.PrevVersionNumber) || !equalInt(g.VersionNumber, w.VersionNumber) {
				t.Errorf("release %s change[%d] after rebuild = %+v, want %+v", id, i, g, w)
			}
		}
	}
	// The kind and label were left alone: they are facts of the cut, not derived.
	if r, _, _ := st.Releases.Changes(ctx, app.ID, r3.ID); r.Kind != models.ReleaseKindRollback || r.SourceReleaseID == nil {
		t.Errorf("rebuild disturbed r3's provenance: %+v", r)
	}
}

func equalStr(a, b *string) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func equalInt(a, b *int) bool    { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
