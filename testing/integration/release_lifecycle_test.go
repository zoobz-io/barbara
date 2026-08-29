//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/jobs"
)

// drainAndRefresh runs the outbox through the pipeline and forces OpenSearch to
// make the writes visible — the site-facing reads are term queries, so a write
// isn't searchable until the index refreshes.
func drainAndRefresh(t *testing.T, st *stores.Stores, pipeline *jobs.Pipeline, provider grub.SearchProvider) {
	t.Helper()
	drainOutbox(t, st, pipeline)
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
}

// mustGet fetches a published page by key and fails if it is absent.
func mustGet(t *testing.T, st *stores.Stores, ctx context.Context, appID, key string) {
	t.Helper()
	if _, err := st.Search.GetPublishedByKey(ctx, appID, key); err != nil {
		t.Fatalf("site should serve %q: %v", key, err)
	}
}

// mustMiss asserts a key is NOT served by the live site.
func mustMiss(t *testing.T, st *stores.Stores, ctx context.Context, appID, key string) {
	t.Helper()
	if _, err := st.Search.GetPublishedByKey(ctx, appID, key); !errors.Is(err, stores.ErrNotFound) {
		t.Fatalf("site should not serve %q: %v", key, err)
	}
}

// lifeTree is the app and tree a lifecycle test drives.
type lifeTree struct {
	appID  string
	refID  string // guides/api/ref.md
	guides string // collection id
	api    string // collection id
}

// lifeApp builds an app with a small tree — guides/api/ref.md, guides/intro.md,
// readme.md — each with one saved version.
func lifeApp(t *testing.T, st *stores.Stores, ctx context.Context) lifeTree {
	t.Helper()
	app, err := st.Apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	guides, _ := st.Collections.Create(ctx, app.ID, nil, "guides")
	api, _ := st.Collections.Create(ctx, app.ID, &guides.ID, "api")

	ref, _ := st.Documents.Create(ctx, app.ID, &api.ID, "ref.md") // guides/api/ref.md
	if _, err := st.Versions.Save(ctx, ref.ID, "the reference", 0); err != nil {
		t.Fatalf("save ref: %v", err)
	}
	intro, _ := st.Documents.Create(ctx, app.ID, &guides.ID, "intro.md") // guides/intro.md
	if _, err := st.Versions.Save(ctx, intro.ID, "an introduction", 0); err != nil {
		t.Fatalf("save intro: %v", err)
	}
	readme, _ := st.Documents.Create(ctx, app.ID, nil, "readme.md") // readme.md
	if _, err := st.Versions.Save(ctx, readme.ID, "the front page", 0); err != nil {
		t.Fatalf("save readme: %v", err)
	}
	return lifeTree{appID: app.ID, refID: ref.ID, guides: guides.ID, api: api.ID}
}

// TestLifecycle_FullPathToSite walks the whole write path — app, collection tree,
// documents, a release — and confirms the site-facing surface serves the current
// release by key, by folder, by full text, and only within the app.
func TestLifecycle_FullPathToSite(t *testing.T) {
	st, pipeline, provider := projectionFixture(t)
	ctx := tenantCtx(testTenant)
	tree := lifeApp(t, st, ctx)
	appID := tree.appID

	if _, err := st.Releases.Cut(ctx, appID); err != nil {
		t.Fatalf("cut r1: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider)

	// Page fetch by key.
	mustGet(t, st, ctx, appID, "guides/api/ref.md")
	mustGet(t, st, ctx, appID, "guides/intro.md")
	mustGet(t, st, ctx, appID, "readme.md")

	// Folder listing by parent_path: guides holds intro.md (ref.md is under
	// guides/api), guides/api holds ref.md, the root holds readme.md.
	if docs, _, _ := st.Search.ListFolder(ctx, appID, "guides", 50, 0); len(docs) != 1 || docs[0].Key != "guides/intro.md" {
		t.Errorf("guides folder = %+v, want [guides/intro.md]", docs)
	}
	if docs, _, _ := st.Search.ListFolder(ctx, appID, "guides/api", 50, 0); len(docs) != 1 || docs[0].Key != "guides/api/ref.md" {
		t.Errorf("guides/api folder = %+v, want [guides/api/ref.md]", docs)
	}

	// Full-text search within the app.
	if hits, _, _ := st.Search.Search(ctx, appID, "introduction", 10, 0); len(hits) != 1 || hits[0].Key != "guides/intro.md" {
		t.Errorf("search = %+v, want the intro", hits)
	}

	// App scoping: a different app serves none of it.
	other, _ := st.Apps.Create(ctx, "other")
	mustMiss(t, st, ctx, other.ID, "readme.md")
}

// TestLifecycle_AuthoringMoveIsNotLiveUntilRecut is the atomicity guarantee:
// moving a document in the tree changes nothing on the live site until the next
// release is cut — no mid-edit 404s, no live URL moving under a reader.
func TestLifecycle_AuthoringMoveIsNotLiveUntilRecut(t *testing.T) {
	st, pipeline, provider := projectionFixture(t)
	ctx := tenantCtx(testTenant)
	tree := lifeApp(t, st, ctx)
	appID, refID := tree.appID, tree.refID

	if _, err := st.Releases.Cut(ctx, appID); err != nil {
		t.Fatalf("cut r1: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider)
	mustGet(t, st, ctx, appID, "guides/api/ref.md")

	// Move ref.md up to guides/ in authoring — no release cut.
	if _, err := st.Documents.Move(ctx, appID, refID, &tree.guides, "ref.md"); err != nil {
		t.Fatalf("move: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider) // nothing was enqueued, but prove it

	// The live site is unchanged: still the OLD path, and the new path is not live.
	mustGet(t, st, ctx, appID, "guides/api/ref.md")
	mustMiss(t, st, ctx, appID, "guides/ref.md")

	// Cut a new release: now the site follows the move — new path live, old gone.
	if _, err := st.Releases.Cut(ctx, appID); err != nil {
		t.Fatalf("cut r2: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider)
	mustGet(t, st, ctx, appID, "guides/ref.md")
	mustMiss(t, st, ctx, appID, "guides/api/ref.md")
}

// TestLifecycle_RollbackRevertsSite confirms rollback cuts a NEW forward release
// (numbers never go backward) and the live site reverts to that release's paths.
func TestLifecycle_RollbackRevertsSite(t *testing.T) {
	st, pipeline, provider := projectionFixture(t)
	ctx := tenantCtx(testTenant)
	tree := lifeApp(t, st, ctx)
	appID, refID := tree.appID, tree.refID

	r1, _ := st.Releases.Cut(ctx, appID)
	drainAndRefresh(t, st, pipeline, provider)

	// Move ref.md and cut r2 — the site now serves the new path.
	if _, err := st.Documents.Move(ctx, appID, refID, &tree.guides, "ref.md"); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := st.Releases.Cut(ctx, appID); err != nil {
		t.Fatalf("cut r2: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider)
	mustGet(t, st, ctx, appID, "guides/ref.md")

	// Roll back to r1: a new forward release restores the old path.
	r3, err := st.Releases.Rollback(ctx, appID, r1.ID)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if r3.Number <= r1.Number {
		t.Errorf("rollback release number %d did not move forward past r1 (%d)", r3.Number, r1.Number)
	}
	drainAndRefresh(t, st, pipeline, provider)
	mustGet(t, st, ctx, appID, "guides/api/ref.md")
	mustMiss(t, st, ctx, appID, "guides/ref.md")

	// The release history is a straight line, newest first.
	list, _ := st.Releases.List(ctx, appID, 50, 0)
	if len(list) != 3 || list[0].Number != 3 || list[2].Number != 1 {
		t.Errorf("release numbers = %+v, want 3,2,1", list)
	}
}

// TestLifecycle_UnpublishThenDelete ties the delete rules to the site: unpublish
// drops a page from the live index (a release without it), and the now-unlive
// document — still referenced by an earlier release — soft-deletes, freeing its
// key while keeping its versions.
func TestLifecycle_UnpublishThenDelete(t *testing.T) {
	st, pipeline, provider := projectionFixture(t)
	ctx := tenantCtx(testTenant)
	tree := lifeApp(t, st, ctx)
	appID, refID := tree.appID, tree.refID

	if _, err := st.Releases.Cut(ctx, appID); err != nil {
		t.Fatalf("cut r1: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider)
	mustGet(t, st, ctx, appID, "guides/api/ref.md")

	// Unpublish ref.md: a new release without its path. The live site drops it.
	if _, err := st.Unpublish(ctx, refID); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	drainAndRefresh(t, st, pipeline, provider)
	mustMiss(t, st, ctx, appID, "guides/api/ref.md")

	// It is absent from the current release but referenced by r1, so deleting it
	// soft-deletes: the row survives with deleted_at set, and the key is freed.
	if err := st.Documents.Delete(ctx, refID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	soft, err := st.Documents.Get(ctx, refID)
	if err != nil {
		t.Fatalf("soft-deleted row should survive (referenced by r1): %v", err)
	}
	if soft.DeletedAt == nil {
		t.Error("soft delete did not set deleted_at")
	}
	// The freed key can be reused by a fresh document.
	if _, err := st.Documents.Create(ctx, appID, &tree.api, "ref.md"); err != nil {
		t.Errorf("key not freed after soft delete: %v", err)
	}
}
