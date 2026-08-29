//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/boot"
	"github.com/zoobz-io/barbara/internal/jobs"
)

// e2eFixture wires the stores aggregate over BOTH real backends — Postgres and a
// freshly-mapped OpenSearch documents index. Unlike publishFixture (which stubs
// the search side, since publishing only enqueues), this is the full async path:
// publish commits the pointer move + outbox row to Postgres, and the caller
// drains the outbox through the real jobs pipeline to land the OpenSearch write —
// exactly as the runner does in production. It returns the aggregate and the
// provider handle (for Refresh, which forces OpenSearch to make writes visible).
func e2eFixture(t *testing.T) (*stores.Stores, grub.SearchProvider) {
	t.Helper()
	db := pgDB(t) // resets the sum catalog; skips when Postgres is absent
	addr := osAddr(t)
	provider := osProvider(t)
	ctx := context.Background()

	// A fresh, explicitly-mapped index (keyword key/tags, analyzed content) so
	// term and full-text queries behave.
	deleteIndex(t, addr, "documents")
	if err := boot.EnsureIndices(ctx, addr); err != nil {
		t.Fatalf("ensure indices: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM jobs")
		_, _ = db.Exec("DELETE FROM versions")
		_, _ = db.Exec("DELETE FROM documents")
		_ = db.Close()
		deleteIndex(t, addr, "documents")
	})

	return stores.New(db, astqlpg.New(), provider), provider
}

// drainOutbox claims every pending job and drives each through the pipeline,
// recording done/failed — the same loop the runner runs on its ticker, executed
// once, synchronously. Returns the number of jobs processed.
func drainOutbox(t *testing.T, st *stores.Stores, pipeline *jobs.Pipeline) int {
	t.Helper()
	ctx := context.Background() // the outbox is tenant-agnostic operational machinery
	claimed, err := st.Jobs.ClaimPending(ctx, 50)
	if err != nil {
		t.Fatalf("claim pending: %v", err)
	}
	for _, j := range claimed {
		if procErr := pipeline.Process(ctx, j); procErr != nil {
			if err := st.Jobs.MarkFailed(ctx, j.ID, procErr.Error()); err != nil {
				t.Fatalf("mark failed: %v", err)
			}
			continue
		}
		if err := st.Jobs.MarkDone(ctx, j.ID); err != nil {
			t.Fatalf("mark done: %v", err)
		}
	}
	return len(claimed)
}

// TestEndToEnd_PublishThroughPipelineIsSearchable is the headline acceptance
// path: create → save → publish → the jobs pipeline lands the OpenSearch write →
// the site-facing surface serves it by key and by full-text search. It also
// pins the eventual-consistency window: right after publish, before the outbox
// drains, the site surface does NOT yet serve the document.
func TestEndToEnd_PublishThroughPipelineIsSearchable(t *testing.T) {
	st, provider := e2eFixture(t)
	ctx := tenantCtx(testTenant)

	doc, err := st.Documents.Create(ctx, "guides/install.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	v, err := st.Versions.Save(ctx, doc.ID, "how to install the widget")
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err := st.Publish(ctx, doc.ID, v.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// pg is committed but the OpenSearch write is still an outbox job: the site
	// surface lags authoring, so the document is not yet served.
	if _, err := st.Search.GetPublishedByKey(ctx, "guides/install.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("pre-drain get = %v, want ErrNotFound (write has not landed yet)", err)
	}

	// Drain the outbox through the real pipeline; the OpenSearch write lands.
	pipeline := jobs.NewPipeline(st.Search, 3, 10*time.Millisecond)
	if n := drainOutbox(t, st, pipeline); n != 1 {
		t.Fatalf("drained %d jobs, want 1 (the index job)", n)
	}
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// Site-facing get-by-key now serves the merged projection.
	got, err := st.Search.GetPublishedByKey(ctx, "guides/install.md")
	if err != nil {
		t.Fatalf("get by key after drain: %v", err)
	}
	if got.DocumentID != doc.ID || got.Content != "how to install the widget" {
		t.Errorf("served projection = %+v, want doc %s with the version content", got, doc.ID)
	}

	// And full-text search over the analyzed content finds it.
	hits, total, err := st.Search.Search(ctx, "install", 10, 0)
	if err != nil {
		t.Fatalf("full-text search: %v", err)
	}
	if total != 1 || len(hits) != 1 || hits[0].DocumentID != doc.ID {
		t.Errorf("full-text 'install' = %d hits, want 1 (%s)", total, doc.ID)
	}
}

// TestEndToEnd_UnpublishRemovesFromSearch proves the removal half of the
// invariant: once a published document is unpublished and the outbox drains, the
// live OpenSearch entry is gone and the site surface no longer serves it.
func TestEndToEnd_UnpublishRemovesFromSearch(t *testing.T) {
	st, provider := e2eFixture(t)
	ctx := tenantCtx(testTenant)
	pipeline := jobs.NewPipeline(st.Search, 3, 10*time.Millisecond)

	doc, err := st.Documents.Create(ctx, "guides/config.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	v, err := st.Versions.Save(ctx, doc.ID, "configure the widget")
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err := st.Publish(ctx, doc.ID, v.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	drainOutbox(t, st, pipeline)
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh after publish: %v", err)
	}
	if _, err := st.Search.GetPublishedByKey(ctx, "guides/config.md"); err != nil {
		t.Fatalf("precondition: document should be served after publish: %v", err)
	}

	// Unpublish enqueues a delete; draining it removes the live entry.
	if _, err := st.Unpublish(ctx, doc.ID); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	if n := drainOutbox(t, st, pipeline); n != 1 {
		t.Fatalf("drained %d jobs after unpublish, want 1 (the delete job)", n)
	}
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh after unpublish: %v", err)
	}

	if _, err := st.Search.GetPublishedByKey(ctx, "guides/config.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("post-unpublish get = %v, want ErrNotFound (live entry removed)", err)
	}
}

// flakyWriter wraps the real search write side and fails the first failsLeft
// Index calls with a transient error before delegating — a stand-in for an
// OpenSearch cluster that is briefly unavailable.
type flakyWriter struct {
	inner     jobs.IndexWriter
	failsLeft int
	attempts  int
}

func (f *flakyWriter) Index(ctx context.Context, documentID string, payload []byte) error {
	f.attempts++
	if f.failsLeft > 0 {
		f.failsLeft--
		return errors.New("transient OpenSearch failure")
	}
	return f.inner.Index(ctx, documentID, payload)
}

func (f *flakyWriter) Delete(ctx context.Context, documentID string) error {
	return f.inner.Delete(ctx, documentID)
}

// TestEndToEnd_TransientOSFailureRetriesAndLands proves the retry invariant: a
// transient OpenSearch write failure is retried by the pipeline's backoff and
// eventually lands, so the document ends up served — the job never turns
// terminal-failed on a blip.
func TestEndToEnd_TransientOSFailureRetriesAndLands(t *testing.T) {
	st, provider := e2eFixture(t)
	ctx := tenantCtx(testTenant)

	doc, err := st.Documents.Create(ctx, "guides/retry.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	v, err := st.Versions.Save(ctx, doc.ID, "retry lands eventually")
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err := st.Publish(ctx, doc.ID, v.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// The first write attempt fails; the pipeline (3 attempts) retries and the
	// second lands against the real cluster.
	flaky := &flakyWriter{inner: st.Search, failsLeft: 1}
	pipeline := jobs.NewPipeline(flaky, 3, 10*time.Millisecond)
	drainOutbox(t, st, pipeline)

	if flaky.attempts < 2 {
		t.Errorf("write attempts = %d, want >= 2 (a transient failure then a retry)", flaky.attempts)
	}
	if err := provider.Refresh(context.Background(), "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// The retry landed, so the site surface serves the document.
	if _, err := st.Search.GetPublishedByKey(ctx, "guides/retry.md"); err != nil {
		t.Errorf("get after retry = %v, want the document served (retry should have landed)", err)
	}
	// And the job settled done, not failed: nothing pending remains.
	remaining, err := st.Jobs.ClaimPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("claim pending after drain: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("pending jobs after successful retry = %d, want 0 (job marked done)", len(remaining))
	}
}
