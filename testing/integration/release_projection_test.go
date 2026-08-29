//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/boot"
	"github.com/zoobz-io/barbara/internal/jobs"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// projectionFixture wires the stores over real Postgres and OpenSearch with a
// fresh index, plus the jobs pipeline, and clears every table in FK order.
func projectionFixture(t *testing.T) (*stores.Stores, *jobs.Pipeline, grub.SearchProvider) {
	t.Helper()
	db := pgDB(t)
	addr := osAddr(t)
	provider := osProvider(t)
	deleteIndex(t, addr, "documents")
	if err := boot.EnsureIndices(context.Background(), addr); err != nil {
		t.Fatalf("ensure indices: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("UPDATE apps SET current_release_id = NULL")
		_, _ = db.Exec("DELETE FROM release_entries")
		_, _ = db.Exec("DELETE FROM releases")
		_, _ = db.Exec("DELETE FROM jobs")
		_, _ = db.Exec("DELETE FROM documents")
		_, _ = db.Exec("DELETE FROM collections")
		_, _ = db.Exec("DELETE FROM apps")
		_ = db.Close()
		deleteIndex(t, addr, "documents")
	})
	st := stores.New(db, astqlpg.New(), provider, testkit.NewBucketProvider())
	return st, newPipeline(st), provider
}

// TestReleaseProjection_AddChangeRemove proves a release cut diffs into the live
// index: added and changed paths appear (with app_id and parent_path
// materialized), a removed path leaves, and the index matches the current
// release.
func TestReleaseProjection_AddChangeRemove(t *testing.T) {
	st, pipeline, provider := projectionFixture(t)
	ctx := tenantCtx(testTenant)
	drain := func() {
		t.Helper()
		drainOutbox(t, st, pipeline)
		if err := provider.Refresh(context.Background(), "documents"); err != nil {
			t.Fatalf("refresh: %v", err)
		}
	}

	app, err := st.Apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	guides, _ := st.Collections.Create(ctx, app.ID, nil, "guides")
	a, _ := st.Documents.Create(ctx, app.ID, &guides.ID, "a.md") // key guides/a.md
	if _, err := st.Versions.Save(ctx, a.ID, "a one", 0); err != nil {
		t.Fatalf("save a v1: %v", err)
	}

	// Cut r1 → a is ADDED. Drain the outbox and confirm it landed with the
	// materialized fields.
	if _, err := st.Releases.Cut(ctx, app.ID); err != nil {
		t.Fatalf("cut r1: %v", err)
	}
	drain()

	got, err := st.Search.GetPublishedByKey(ctx, "guides/a.md")
	if err != nil {
		t.Fatalf("a should be indexed after r1: %v", err)
	}
	if got.Content != "a one" || got.AppID != app.ID || got.ParentPath != "guides" {
		t.Errorf("projected a = content:%q app:%q parent:%q; want 'a one'/%s/guides", got.Content, got.AppID, got.ParentPath, app.ID)
	}

	// Add b, change a; cut r2 (full tree) → a CHANGED, b ADDED.
	b, _ := st.Documents.Create(ctx, app.ID, nil, "b.md")
	if _, err := st.Versions.Save(ctx, b.ID, "b one", 0); err != nil {
		t.Fatalf("save b v1: %v", err)
	}
	if _, err := st.Versions.Save(ctx, a.ID, "a two", 1); err != nil {
		t.Fatalf("save a v2: %v", err)
	}
	if _, err := st.Releases.Cut(ctx, app.ID); err != nil {
		t.Fatalf("cut r2: %v", err)
	}
	drain()

	if got, err := st.Search.GetPublishedByKey(ctx, "guides/a.md"); err != nil || got.Content != "a two" {
		t.Errorf("a should be re-indexed at v2 after r2: content=%q err=%v", got.Content, err)
	}
	if _, err := st.Search.GetPublishedByKey(ctx, "b.md"); err != nil {
		t.Errorf("b should be indexed after r2: %v", err)
	}

	// Cut r3 explicitly WITHOUT b (the unpublish shape) → b REMOVED.
	aHead, _ := st.Versions.Save(ctx, a.ID, "a two", 2) // keep a live at its head
	if _, err := st.Releases.CutWith(ctx, app.ID, []stores.ReleaseEntrySpec{
		{Key: "guides/a.md", DocumentID: a.ID, VersionID: aHead.ID},
	}); err != nil {
		t.Fatalf("cut r3: %v", err)
	}
	drain()

	if _, err := st.Search.GetPublishedByKey(ctx, "b.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("b should be gone from the index after r3 dropped it: %v", err)
	}
	if _, err := st.Search.GetPublishedByKey(ctx, "guides/a.md"); err != nil {
		t.Errorf("a should still be indexed after r3: %v", err)
	}
}
