//go:build testing

package integration

import (
	"testing"

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

	app, err := st.Apps.Create(ctx, "site")
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
	r1, err := st.Releases.Cut(ctx, app.ID)
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
	r2, err := st.Releases.Cut(ctx, app.ID)
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
	r3, err := st.Releases.Rollback(ctx, app.ID, r1.ID)
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
