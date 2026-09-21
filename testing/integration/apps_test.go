//go:build testing

package integration

import (
	"errors"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// TestApps_DeleteCascades proves the delete rules against real Postgres: an
// app with no release goes, and takes its tree, its documents and versions, and
// its asset bookkeeping with it; an app with a release is refused, and nothing
// beneath it moves.
func TestApps_DeleteCascades(t *testing.T) {
	db := pgDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("UPDATE apps SET current_release_id = NULL")
		_, _ = db.Exec("DELETE FROM release_entries")
		_, _ = db.Exec("DELETE FROM releases")
		_, _ = db.Exec("DELETE FROM jobs")
		_, _ = db.Exec("DELETE FROM documents")
		_, _ = db.Exec("DELETE FROM collections")
		_, _ = db.Exec("DELETE FROM asset_folders")
		_, _ = db.Exec("DELETE FROM asset_stats")
		_, _ = db.Exec("DELETE FROM asset_stats_daily")
		_, _ = db.Exec("DELETE FROM asset_bookkeeping")
		_, _ = db.Exec("DELETE FROM apps")
		_ = db.Close()
	})
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	ctx := tenantCtx(testTenant)

	count := func(table, appID string) int {
		t.Helper()
		var n int
		if err := db.QueryRowx("SELECT count(*) FROM "+table+" WHERE app_id = $1", appID).Scan(&n); err != nil {
			t.Fatalf("counting %s: %v", table, err)
		}
		return n
	}
	countVersions := func(appID string) int {
		t.Helper()
		var n int
		if err := db.QueryRowx(
			"SELECT count(*) FROM versions v JOIN documents d ON d.id = v.document_id WHERE d.app_id = $1",
			appID).Scan(&n); err != nil {
			t.Fatalf("counting versions: %v", err)
		}
		return n
	}

	// A populated app that was never released: a folder, a page in it with a
	// saved version, and an explicit asset folder (a bookkeeping row with no
	// object behind it).
	app := seedApp(t, st, ctx)
	col, err := st.Collections.Create(ctx, app.ID, nil, "guides")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	doc, err := st.Documents.Create(ctx, app.ID, &col.ID, "intro.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if _, err := st.Versions.Save(ctx, doc.ID, "# intro", 0); err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err := st.Assets.CreateFolder(ctx, app.ID, "images"); err != nil {
		t.Fatalf("create asset folder: %v", err)
	}
	if n := count("asset_folders", app.ID); n == 0 {
		t.Fatal("explicit folder left no bookkeeping row; the cascade has nothing to prove")
	}

	if err := st.Apps.Delete(ctx, app.ID); err != nil {
		t.Fatalf("delete never-released app: %v", err)
	}
	for _, table := range []string{"apps", "collections", "documents", "asset_folders", "asset_stats", "asset_stats_daily", "asset_bookkeeping"} {
		if n := count(table, app.ID); n != 0 {
			t.Errorf("%s still holds %d rows for the deleted app", table, n)
		}
	}
	if n := countVersions(app.ID); n != 0 {
		t.Errorf("versions still holds %d rows for the deleted app", n)
	}

	// A released app is refused, and stays whole — including the bookkeeping,
	// which must not be what a delete trips over.
	kept := seedApp(t, st, ctx)
	keptDoc, err := st.Documents.Create(ctx, kept.ID, nil, "home.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if _, err := st.Versions.Save(ctx, keptDoc.ID, "# home", 0); err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err := st.Assets.CreateFolder(ctx, kept.ID, "images"); err != nil {
		t.Fatalf("create asset folder: %v", err)
	}
	if _, err := st.Releases.Cut(ctx, kept.ID, ""); err != nil {
		t.Fatalf("cut: %v", err)
	}
	if err := st.Apps.Delete(ctx, kept.ID); !errors.Is(err, stores.ErrAppHasReleases) {
		t.Fatalf("delete released app = %v, want ErrAppHasReleases", err)
	}
	for _, table := range []string{"apps", "documents", "asset_folders", "releases"} {
		if n := count(table, kept.ID); n == 0 {
			t.Errorf("%s lost its rows on a refused delete", table)
		}
	}
}
