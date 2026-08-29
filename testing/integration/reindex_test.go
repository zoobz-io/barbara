//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/boot"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// reindexFixture builds the aggregate over real Postgres AND a real OpenSearch
// index (reindex writes projections straight through the search store), with
// published documents under two tenants plus one draft that must be excluded.
func reindexFixture(t *testing.T) (*stores.Stores, *models.App, *models.App) {
	t.Helper()
	db := pgDB(t)
	addr := osAddr(t)
	provider := osProvider(t)
	ctx := context.Background()

	deleteIndex(t, addr, "documents")
	if err := boot.EnsureIndices(ctx, addr); err != nil {
		t.Fatalf("ensure indices: %v", err)
	}
	t.Cleanup(func() {
		deleteIndex(t, addr, "documents")
		_, _ = db.Exec("UPDATE apps SET current_release_id = NULL")
		_, _ = db.Exec("DELETE FROM release_entries")
		_, _ = db.Exec("DELETE FROM releases")
		_, _ = db.Exec("DELETE FROM jobs")
		_, _ = db.Exec("DELETE FROM versions")
		_, _ = db.Exec("DELETE FROM documents")
		_, _ = db.Exec("DELETE FROM collections")
		_, _ = db.Exec("DELETE FROM apps")
		_ = db.Close()
	})

	st := stores.New(db, astqlpg.New(), provider, testkit.NewBucketProvider())

	// t1: one released doc — cut BEFORE the draft exists, so the release holds
	// exactly it.
	t1 := tenantCtx(testTenant)
	app1 := seedApp(t, st, t1)
	a, err := seedDoc(st, t1, app1.ID, "guides/a.md")
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	_, err = st.Versions.Save(t1, a.ID, "# alpha", 0)
	if err != nil {
		t.Fatalf("save a version: %v", err)
	}
	if _, err := st.Releases.Cut(t1, app1.ID); err != nil {
		t.Fatalf("cut r1 for app1: %v", err)
	}
	// Rename a AFTER the cut: the authoring key moves to renamed.md, but the
	// release recorded guides/a.md — the reindex must serve the entry key.
	if _, err := st.Documents.Move(t1, app1.ID, a.ID, nil, "renamed.md"); err != nil {
		t.Fatalf("rename a: %v", err)
	}

	// t1: a draft (saved after the cut, in no release) — must not be reindexed.
	d, err := seedDoc(st, t1, app1.ID, "draft.md")
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if _, err := st.Versions.Save(t1, d.ID, "# unpublished", 0); err != nil {
		t.Fatalf("save draft version: %v", err)
	}

	// t2: one released doc — proves the reindex crosses tenants and apps.
	t2 := tenantCtx(otherTenant)
	app2 := seedApp(t, st, t2)
	b, err := seedDoc(st, t2, app2.ID, "guides/b.md")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	_, err = st.Versions.Save(t2, b.ID, "# beta", 0)
	if err != nil {
		t.Fatalf("save b version: %v", err)
	}
	if _, err := st.Releases.Cut(t2, app2.ID); err != nil {
		t.Fatalf("cut r1 for app2: %v", err)
	}

	// The cuts only enqueued outbox jobs (never drained); the index is still
	// empty. Reindex must rebuild it from the current releases alone.
	return st, app1, app2
}

func TestReindex_RebuildsPublishedSetFromPostgres(t *testing.T) {
	st, app1, app2 := reindexFixture(t)
	provider := osProvider(t)
	ctx := context.Background()

	n, err := st.Reindex(ctx)
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if n != 2 {
		t.Errorf("reindexed %d documents, want 2 (the published set across both tenants)", n)
	}
	if err := provider.Refresh(ctx, "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// The site-facing surface now serves exactly the published set.
	t1 := tenantCtx(testTenant)
	if doc, err := st.Search.GetPublishedByKey(t1, app1.ID, "guides/a.md"); err != nil || doc.Content != "# alpha" {
		t.Errorf("t1 guides/a.md = %+v, %v; want the published projection", doc, err)
	}
	// The draft was excluded.
	if _, err := st.Search.GetPublishedByKey(t1, app1.ID, "draft.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("draft.md = %v, want ErrNotFound (drafts are not projected)", err)
	}
	// The post-cut rename did not leak: the release-recorded path serves, the
	// new authoring key does not.
	if _, err := st.Search.GetPublishedByKey(t1, app1.ID, "renamed.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("renamed.md = %v, want ErrNotFound (authoring key is not the serving key)", err)
	}
	// Each tenant sees only its own published document.
	if _, total, err := st.Search.Enumerate(t1, app1.ID, "", 50, 0); err != nil || total != 1 {
		t.Errorf("t1 enumerate total = %d, %v; want 1", total, err)
	}
	t2 := tenantCtx(otherTenant)
	if doc, err := st.Search.GetPublishedByKey(t2, app2.ID, "guides/b.md"); err != nil || doc.Content != "# beta" {
		t.Errorf("t2 guides/b.md = %+v, %v; want the published projection", doc, err)
	}
}

func TestReindex_Idempotent(t *testing.T) {
	st, app1, _ := reindexFixture(t)
	provider := osProvider(t)
	ctx := context.Background()

	if _, err := st.Reindex(ctx); err != nil {
		t.Fatalf("first reindex: %v", err)
	}
	n, err := st.Reindex(ctx)
	if err != nil {
		t.Fatalf("second reindex: %v", err)
	}
	if n != 2 {
		t.Errorf("second reindex projected %d, want 2 (same set)", n)
	}
	if err := provider.Refresh(ctx, "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// Re-running converges to one live entry per published document, not duplicates.
	if _, total, err := st.Search.Enumerate(tenantCtx(testTenant), app1.ID, "", 50, 0); err != nil || total != 1 {
		t.Errorf("t1 enumerate total after re-run = %d, %v; want 1 (upsert, no duplicates)", total, err)
	}
}
