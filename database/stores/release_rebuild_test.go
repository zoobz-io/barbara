//go:build testing

package stores

import (
	"testing"

	"github.com/zoobz-io/grub/mockdb"

	"github.com/zoobz-io/barbara/database/models"
)

// releaseRows is a releases result set of several rows, oldest first.
func releaseRows(ids ...string) *mockdb.RowData {
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, testApp, testTenant, int64(i + 1), testUser, models.ReleaseKindCut}
	}
	return &mockdb.RowData{
		Columns: []string{"id", "app_id", "tenant_id", "number", "created_by", "kind"},
		Rows:    rows,
	}
}

// rebuildApp walks an app's releases oldest first: for each, load its entries,
// clear its change rows, write the diff against the previous release, and
// write its counts. The first release's entry is an addition; the second, at
// a new version of the same document, a change.
func TestReleases_RebuildApp_RecomputesChangesAndCounts(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(releaseRows("r-1", "r-2"))             // releases, oldest first
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-1")) // r-1 entries
	cfg.PushRowData(changeRow("a.md", "d-1", "added"))     // r-1: change insert (delete needs no rows)
	cfg.PushRowData(releaseRow("r-1", 1))                  // r-1: counts update RETURNING
	cfg.PushRowData(releaseEntryRow("a.md", "d-1", "v-2")) // r-2 entries
	cfg.PushRowData(changeRow("a.md", "d-1", "changed"))   // r-2: change insert
	cfg.PushRowData(releaseRow("r-2", 2))                  // r-2: counts update

	n, err := st.Releases.rebuildApp(tenantCtx(), testApp)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if n != 2 {
		t.Errorf("rebuilt = %d, want 2", n)
	}

	list := queryAt(t, capture, 0)
	wantSQL(t, list, `FROM "releases"`, `"app_id" = ?`, `ORDER BY "number" ASC`)
	notSQL(t, list, `"tenant_id"`) // operational: every tenant's releases

	// Per release: entries, DELETE changes, INSERT changes, UPDATE counts.
	clear1 := queryAt(t, capture, 2)
	wantSQL(t, clear1, `DELETE FROM "release_changes"`, `"release_id" = ?`)
	wantArg(t, clear1, "r-1")
	add := queryAt(t, capture, 3)
	wantSQL(t, add, `INSERT INTO "release_changes"`)
	wantArg(t, add, models.ChangeAdded)
	counts1 := queryAt(t, capture, 4)
	wantSQL(t, counts1, `UPDATE "releases" SET`, `"entry_count" = ?`, `"added" = ?`, `"id" = ?`)
	wantArg(t, counts1, "r-1")

	change := queryAt(t, capture, 7)
	wantSQL(t, change, `INSERT INTO "release_changes"`)
	wantArg(t, change, models.ChangeChanged)
	wantArg(t, change, "v-1") // the previous version
	wantArg(t, change, "v-2")
	counts2 := queryAt(t, capture, 8)
	wantSQL(t, counts2, `UPDATE "releases" SET`, `"changed" = ?`)
	wantArg(t, counts2, "r-2")
}

// An app with no releases rebuilds nothing and touches no other table.
func TestReleases_RebuildApp_Empty(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(noRows())

	n, err := st.Releases.rebuildApp(tenantCtx(), testApp)
	if err != nil || n != 0 {
		t.Fatalf("rebuild of an empty app = %d, %v; want 0, nil", n, err)
	}
	if len(capture.Queries) != 1 {
		t.Errorf("queries = %d, want only the releases listing", len(capture.Queries))
	}
}
