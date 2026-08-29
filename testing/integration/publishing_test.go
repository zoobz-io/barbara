//go:build testing

package integration

import (
	"encoding/json"
	"errors"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// publishFixture builds the aggregate over real Postgres (the search provider is
// a stub — publishing enqueues jobs and never calls OpenSearch) plus a document
// with one saved version.
func publishFixture(t *testing.T) (*stores.Stores, string, string) {
	t.Helper()
	db := pgDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM jobs")
		_, _ = db.Exec("DELETE FROM versions")
		_, _ = db.Exec("DELETE FROM documents")
		_ = db.Close()
	})
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	ctx := tenantCtx(testTenant)
	doc, err := st.Documents.Create(ctx, "publishable.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	v, err := st.Versions.Save(ctx, doc.ID, "# content")
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	return st, doc.ID, v.ID
}

func TestPublish_MovesPointerAndEnqueuesProjection(t *testing.T) {
	st, docID, versionID := publishFixture(t)
	ctx := tenantCtx(testTenant)

	updated, err := st.Publish(ctx, docID, versionID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if updated.PublishedVersionID == nil || *updated.PublishedVersionID != versionID {
		t.Fatalf("published_version_id = %v, want %s", updated.PublishedVersionID, versionID)
	}

	// An index job was enqueued, pending, carrying the merged projection.
	claimed, err := st.Jobs.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("enqueued %d jobs, want 1", len(claimed))
	}
	job := claimed[0]
	if job.Operation != models.JobIndex || job.DocumentID != docID {
		t.Errorf("unexpected job: op=%s doc=%s", job.Operation, job.DocumentID)
	}
	var idx models.DocumentIndex
	if err := json.Unmarshal(job.Payload, &idx); err != nil {
		t.Fatalf("projection is not valid json: %v", err)
	}
	if idx.DocumentID != docID || idx.VersionID != versionID || idx.Content != "# content" || idx.Key != "publishable.md" {
		t.Errorf("projection did not merge doc+version: %+v", idx)
	}
}

func TestUnpublish_ClearsPointerAndEnqueuesDelete(t *testing.T) {
	st, docID, versionID := publishFixture(t)
	ctx := tenantCtx(testTenant)

	if _, err := st.Publish(ctx, docID, versionID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := st.Jobs.ClaimPending(ctx, 10); err != nil { // drain the index job
		t.Fatalf("drain: %v", err)
	}

	updated, err := st.Unpublish(ctx, docID)
	if err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	if updated.PublishedVersionID != nil {
		t.Errorf("published_version_id = %v, want nil", *updated.PublishedVersionID)
	}

	claimed, err := st.Jobs.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 || claimed[0].Operation != models.JobDelete || claimed[0].DocumentID != docID {
		t.Errorf("expected one delete job for the document, got %+v", claimed)
	}
}

func TestRollback_RepublishesOlderVersion(t *testing.T) {
	st, docID, v1 := publishFixture(t)
	ctx := tenantCtx(testTenant)

	v2, err := st.Versions.Save(ctx, docID, "# newer")
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if _, err := st.Publish(ctx, docID, v2.ID); err != nil {
		t.Fatalf("publish v2: %v", err)
	}

	// Roll back to v1.
	updated, err := st.Rollback(ctx, docID, v1)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if updated.PublishedVersionID == nil || *updated.PublishedVersionID != v1 {
		t.Errorf("published_version_id = %v, want %s (v1)", updated.PublishedVersionID, v1)
	}
}

func TestPublish_RejectsForeignVersion(t *testing.T) {
	st, docID, _ := publishFixture(t)
	ctx := tenantCtx(testTenant)

	// A version belonging to a different document.
	other, err := st.Documents.Create(ctx, "other.md")
	if err != nil {
		t.Fatalf("create other doc: %v", err)
	}
	foreign, err := st.Versions.Save(ctx, other.ID, "x")
	if err != nil {
		t.Fatalf("save foreign version: %v", err)
	}

	if _, err := st.Publish(ctx, docID, foreign.ID); !errors.Is(err, stores.ErrVersionMismatch) {
		t.Errorf("publish foreign version = %v, want ErrVersionMismatch", err)
	}
	if _, err := st.Publish(ctx, docID, "44444444-0000-0000-0000-000000000004"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("publish missing version = %v, want ErrNotFound", err)
	}
}
