//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/jobs"
)

// These are behavioural scenarios: they drive the features the way an author and
// a reader actually use them — edit and republish, roll back, retag, delete —
// and assert what the *site-facing surface serves* after each change lands
// through the real jobs pipeline. Where the store-level tests stop at the outbox
// (the right job was enqueued), these follow the write all the way to what a
// reader sees, so they document how the system behaves in use, not just that a
// query was built.

// newPipeline builds the real OpenSearch-write pipeline over the search store,
// with a short backoff so retry paths don't dawdle.
func newPipeline(st *stores.Stores) *jobs.Pipeline {
	return jobs.NewPipeline(st.Search, 3, 10*time.Millisecond)
}

// settle drains the outbox through the pipeline and refreshes OpenSearch, so the
// writes just enqueued are visible to the reads that follow — the test-time
// stand-in for "wait for eventual consistency to converge".
func settle(t *testing.T, st *stores.Stores, provider grub.SearchProvider, pipeline *jobs.Pipeline) {
	t.Helper()
	drainOutbox(t, st, pipeline)
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
}

// authorAndPublish runs the full authoring path for one tenant: create the
// document, save a first version, publish it. It returns the document and
// version ids. It does NOT drain — the caller settles when it wants the write
// visible, so tests can assert the pre-convergence state too.
func authorAndPublish(t *testing.T, st *stores.Stores, tenant, key, content string) (docID, versionID string) {
	t.Helper()
	ctx := tenantCtx(tenant)
	doc, err := st.Documents.Create(ctx, key)
	if err != nil {
		t.Fatalf("create %s: %v", key, err)
	}
	v, err := st.Versions.Save(ctx, doc.ID, content)
	if err != nil {
		t.Fatalf("save %s: %v", key, err)
	}
	if _, err := st.Publish(ctx, doc.ID, v.ID); err != nil {
		t.Fatalf("publish %s: %v", key, err)
	}
	return doc.ID, v.ID
}

// TestScenario_EditAndRepublish_UpdatesServedProjection is the everyday author
// loop: publish, then edit and publish again. The reader must see the newer
// version — same document, same key, new content and version number — and the
// old content must no longer be findable, because the projection is replaced
// (upsert by document id), not appended.
func TestScenario_EditAndRepublish_UpdatesServedProjection(t *testing.T) {
	st, provider := e2eFixture(t)
	pipeline := newPipeline(st)
	ctx := tenantCtx(testTenant)

	docID, _ := authorAndPublish(t, st, testTenant, "guides/widget.md", "the widget points upward")
	settle(t, st, provider, pipeline)

	got, err := st.Search.GetPublishedByKey(ctx, "guides/widget.md")
	if err != nil {
		t.Fatalf("get after first publish: %v", err)
	}
	if got.VersionNumber != 1 || got.Content != "the widget points upward" {
		t.Fatalf("served v1 = {n:%d %q}, want {1 upward}", got.VersionNumber, got.Content)
	}

	// Author edits and republishes.
	v2, err := st.Versions.Save(ctx, docID, "the widget points downward")
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if _, err := st.Publish(ctx, docID, v2.ID); err != nil {
		t.Fatalf("publish v2: %v", err)
	}
	settle(t, st, provider, pipeline)

	got, err = st.Search.GetPublishedByKey(ctx, "guides/widget.md")
	if err != nil {
		t.Fatalf("get after republish: %v", err)
	}
	if got.DocumentID != docID || got.VersionNumber != 2 || got.VersionID != v2.ID {
		t.Errorf("served after republish = {doc:%s n:%d v:%s}, want the same doc at v2 (%s)", got.DocumentID, got.VersionNumber, got.VersionID, v2.ID)
	}
	if got.Content != "the widget points downward" {
		t.Errorf("served content = %q, want the edited content", got.Content)
	}

	// The new content is findable; the replaced content is not — one live entry
	// per document (upsert by id), not an accumulating history.
	if _, total, err := st.Search.Search(ctx, "downward", 10, 0); err != nil || total != 1 {
		t.Errorf("search 'downward' = %d,%v; want 1 hit", total, err)
	}
	if _, total, err := st.Search.Search(ctx, "upward", 10, 0); err != nil || total != 0 {
		t.Errorf("search 'upward' = %d,%v; want 0 (the replaced version's content is no longer served)", total, err)
	}
}

// TestScenario_Rollback_RevertsServedContent proves rollback is observable at
// the surface: after publishing a newer version and rolling back to an older
// one, the reader sees the older version's content again — no copy, just the
// pointer moved and the projection rebuilt.
func TestScenario_Rollback_RevertsServedContent(t *testing.T) {
	st, provider := e2eFixture(t)
	pipeline := newPipeline(st)
	ctx := tenantCtx(testTenant)

	docID, v1 := authorAndPublish(t, st, testTenant, "post.md", "alpha draft")
	v2, err := st.Versions.Save(ctx, docID, "beta rewrite")
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if _, err := st.Publish(ctx, docID, v2.ID); err != nil {
		t.Fatalf("publish v2: %v", err)
	}
	settle(t, st, provider, pipeline)

	if got, err := st.Search.GetPublishedByKey(ctx, "post.md"); err != nil || got.Content != "beta rewrite" {
		t.Fatalf("precondition: served = %+v,%v; want beta", got, err)
	}

	// Roll back to v1.
	if _, err := st.Rollback(ctx, docID, v1); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	settle(t, st, provider, pipeline)

	got, err := st.Search.GetPublishedByKey(ctx, "post.md")
	if err != nil {
		t.Fatalf("get after rollback: %v", err)
	}
	if got.VersionNumber != 1 || got.Content != "alpha draft" {
		t.Errorf("served after rollback = {n:%d %q}, want {1 alpha draft}", got.VersionNumber, got.Content)
	}
	if _, total, err := st.Search.Search(ctx, "beta", 10, 0); err != nil || total != 0 {
		t.Errorf("search 'beta' after rollback = %d,%v; want 0 (rolled-back content not served)", total, err)
	}
}

// TestScenario_TagChangeOnPublished_ReachesSearch proves that tagging a
// published document reprojects all the way to the reader: the tag becomes a
// working filter in Enumerate, and untagging removes it — without unpublishing
// the document.
func TestScenario_TagChangeOnPublished_ReachesSearch(t *testing.T) {
	st, provider := e2eFixture(t)
	pipeline := newPipeline(st)
	ctx := tenantCtx(testTenant)

	docID, _ := authorAndPublish(t, st, testTenant, "howto.md", "step by step")
	settle(t, st, provider, pipeline)

	// Not yet tagged: the tag filter finds nothing.
	if _, total, err := st.Search.Enumerate(ctx, "guide", 50, 0); err != nil || total != 0 {
		t.Fatalf("enumerate 'guide' before tagging = %d,%v; want 0", total, err)
	}

	// Tag it — a metadata change on a published doc reprojects.
	if _, err := st.AddTag(ctx, docID, "guide"); err != nil {
		t.Fatalf("add tag: %v", err)
	}
	settle(t, st, provider, pipeline)

	hits, total, err := st.Search.Enumerate(ctx, "guide", 50, 0)
	if err != nil || total != 1 || hits[0].DocumentID != docID {
		t.Fatalf("enumerate 'guide' after tagging = %d,%v; want 1 (%s)", total, err, docID)
	}
	// Still published, still served by key.
	if got, err := st.Search.GetPublishedByKey(ctx, "howto.md"); err != nil || got.DocumentID != docID {
		t.Errorf("get by key after tagging = %+v,%v; want the doc still served", got, err)
	}

	// Untag it — the filter stops matching, the document stays published.
	if _, err := st.RemoveTag(ctx, docID, "guide"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	settle(t, st, provider, pipeline)
	if _, total, err := st.Search.Enumerate(ctx, "guide", 50, 0); err != nil || total != 0 {
		t.Errorf("enumerate 'guide' after untagging = %d,%v; want 0", total, err)
	}
	if _, err := st.Search.GetPublishedByKey(ctx, "howto.md"); err != nil {
		t.Errorf("doc should still be published after untagging: %v", err)
	}
}

// TestScenario_DraftIsNeverServed proves the draft/published boundary: an
// unpublished document produces no outbox work and is invisible to every
// site-facing read, no matter how many versions it has.
func TestScenario_DraftIsNeverServed(t *testing.T) {
	st, provider := e2eFixture(t)
	pipeline := newPipeline(st)
	ctx := tenantCtx(testTenant)

	doc, err := st.Documents.Create(ctx, "draft.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Versions.Save(ctx, doc.ID, "unpublished words"); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Authoring a draft enqueues no OpenSearch work.
	if n := drainOutbox(t, st, pipeline); n != 0 {
		t.Errorf("draft authoring enqueued %d jobs, want 0", n)
	}
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// And it is served by nothing.
	if _, err := st.Search.GetPublishedByKey(ctx, "draft.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("get draft by key = %v, want ErrNotFound", err)
	}
	if _, total, err := st.Search.Search(ctx, "unpublished", 10, 0); err != nil || total != 0 {
		t.Errorf("search for draft content = %d,%v; want 0", total, err)
	}
	if _, total, err := st.Search.Enumerate(ctx, "", 50, 0); err != nil || total != 0 {
		t.Errorf("enumerate = %d,%v; want 0 (nothing published)", total, err)
	}
}

// TestScenario_TenantIsolation_SameKeyThroughPipeline drives two tenants
// publishing documents under the SAME key and proves isolation holds at the
// serving surface: each tenant's reads see only their own document and content,
// while the cross-tenant admin search sees both.
func TestScenario_TenantIsolation_SameKeyThroughPipeline(t *testing.T) {
	st, provider := e2eFixture(t)
	pipeline := newPipeline(st)

	t1Doc, _ := authorAndPublish(t, st, testTenant, "shared/readme.md", "tenant one handbook")
	t2Doc, _ := authorAndPublish(t, st, otherTenant, "shared/readme.md", "tenant two handbook")
	settle(t, st, provider, pipeline) // drains both tenants' index jobs

	t1 := tenantCtx(testTenant)
	t2 := tenantCtx(otherTenant)

	got1, err := st.Search.GetPublishedByKey(t1, "shared/readme.md")
	if err != nil || got1.DocumentID != t1Doc || got1.Content != "tenant one handbook" {
		t.Errorf("tenant one get = %+v,%v; want its own doc %s", got1, err, t1Doc)
	}
	got2, err := st.Search.GetPublishedByKey(t2, "shared/readme.md")
	if err != nil || got2.DocumentID != t2Doc || got2.Content != "tenant two handbook" {
		t.Errorf("tenant two get = %+v,%v; want its own doc %s", got2, err, t2Doc)
	}

	// Tenant-scoped full-text search never crosses the boundary.
	if hits, total, err := st.Search.Search(t1, "handbook", 10, 0); err != nil || total != 1 || hits[0].DocumentID != t1Doc {
		t.Errorf("tenant one search 'handbook' = %d hits,%v; want just %s", total, err, t1Doc)
	}
	// The admin cross-tenant search sees both.
	if _, total, err := st.Search.SearchAll(context.Background(), "handbook", 10, 0); err != nil || total != 2 {
		t.Errorf("SearchAll 'handbook' = %d,%v; want 2 (both tenants)", total, err)
	}
}

// TestScenario_Enumerate_PaginatesStably publishes several documents and walks
// the site-facing listing in pages: a page is capped at the limit, the total
// reflects the full set, and paging by offset visits every document exactly once
// — no skips, no duplicates. That last property depends on the deterministic
// sort (created_at, then the unique document_id tiebreaker); without it, offset
// paging in filter context is unstable across requests.
func TestScenario_Enumerate_PaginatesStably(t *testing.T) {
	st, provider := e2eFixture(t)
	pipeline := newPipeline(st)
	ctx := tenantCtx(testTenant)

	keys := []string{"a.md", "b.md", "c.md", "d.md", "e.md"}
	for _, k := range keys {
		authorAndPublish(t, st, testTenant, k, "body "+k)
	}
	settle(t, st, provider, pipeline)

	// A limited page is capped at the limit, but the total is the full count.
	page, total, err := st.Search.Enumerate(ctx, "", 2, 0)
	if err != nil {
		t.Fatalf("enumerate limit 2: %v", err)
	}
	if total != int64(len(keys)) {
		t.Errorf("total = %d, want %d (full published set)", total, len(keys))
	}
	if len(page) != 2 {
		t.Errorf("page size = %d, want 2 (capped at the limit)", len(page))
	}

	// Walk every page; each document appears exactly once, and the listing is
	// ordered oldest-first by created_at (the stable sort makes this reliable).
	seen := map[string]bool{}
	var lastCreated time.Time
	for offset := 0; offset < len(keys); offset += 2 {
		p, _, err := st.Search.Enumerate(ctx, "", 2, offset)
		if err != nil {
			t.Fatalf("enumerate page @%d: %v", offset, err)
		}
		for _, d := range p {
			if seen[d.DocumentID] {
				t.Errorf("document %s appeared on more than one page (unstable paging)", d.DocumentID)
			}
			seen[d.DocumentID] = true
			if d.CreatedAt.Before(lastCreated) {
				t.Errorf("listing out of order: %s created %v precedes previous %v", d.DocumentID, d.CreatedAt, lastCreated)
			}
			lastCreated = d.CreatedAt
		}
	}
	if len(seen) != len(keys) {
		t.Errorf("paged through %d distinct docs, want %d (paging skipped some)", len(seen), len(keys))
	}
}

// TestScenario_TerminalOSFailure_ReindexReconciles is the safety net in action:
// when an OpenSearch write fails permanently (retries exhausted), the reader is
// left stale — Postgres has the publish, OpenSearch does not. A full reindex
// (#20) rebuilds the index from Postgres and the reader converges again. This is
// the recovery path the eventual-consistency model rests on.
func TestScenario_TerminalOSFailure_ReindexReconciles(t *testing.T) {
	st, provider := e2eFixture(t)
	ctx := tenantCtx(testTenant)

	authorAndPublish(t, st, testTenant, "durable.md", "must survive a bad write")

	// Every attempt fails: the pipeline exhausts its retries and the job goes
	// terminal-failed. The publish is committed in Postgres, but nothing lands
	// in OpenSearch.
	alwaysFails := &flakyWriter{inner: st.Search, failsLeft: 1_000}
	failing := jobs.NewPipeline(alwaysFails, 3, 10*time.Millisecond)
	drainOutbox(t, st, failing)
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := st.Search.GetPublishedByKey(ctx, "durable.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Fatalf("after terminal failure, get = %v; want ErrNotFound (write never landed)", err)
	}

	// The safety net: reindex rebuilds the index from Postgres alone.
	n, err := st.Reindex(context.Background())
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if n != 1 {
		t.Errorf("reindexed %d documents, want 1 (the published doc)", n)
	}
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh after reindex: %v", err)
	}

	got, err := st.Search.GetPublishedByKey(ctx, "durable.md")
	if err != nil {
		t.Fatalf("get after reindex = %v; want the document served again", err)
	}
	if got.Content != "must survive a bad write" {
		t.Errorf("reconciled content = %q, want the published content", got.Content)
	}
}
