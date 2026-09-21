//go:build testing

package integration

import (
	"context"
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
		resetDB(t, db)
		_ = db.Close()
	})
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	ctx := tenantCtx(testTenant)
	doc, err := seedDoc(st, ctx, seedApp(t, st, ctx).ID, "publishable.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	v, err := st.Versions.Save(ctx, doc.ID, "# content", 0)
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	return st, doc.ID, v.ID
}

func TestPublish_CutsReleaseAndEnqueuesProjection(t *testing.T) {
	st, docID, versionID := publishFixture(t)
	ctx := tenantCtx(testTenant)

	updated, err := st.Publish(ctx, docID, versionID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if status, _ := st.Documents.Status(ctx, updated); status != models.StatusPublished {
		t.Fatalf("status after publish = %q, want published", status)
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
	if status, _ := st.Documents.Status(ctx, updated); status != models.StatusDraft {
		t.Errorf("status after unpublish = %q, want draft", status)
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

	v2, err := st.Versions.Save(ctx, docID, "# newer", 1)
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if _, err := st.Publish(ctx, docID, v2.ID); err != nil {
		t.Fatalf("publish v2: %v", err)
	}

	// Roll back to v1: the current release now serves v1, though the head is v2.
	updated, err := st.Rollback(ctx, docID, v1)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	entry, err := st.Releases.CurrentEntryFor(ctx, updated.AppID, docID)
	if err != nil || entry == nil {
		t.Fatalf("current entry after rollback: entry=%v err=%v", entry, err)
	}
	if entry.VersionID != v1 {
		t.Errorf("current release serves version %s, want %s (v1)", entry.VersionID, v1)
	}
	if status, _ := st.Documents.Status(ctx, updated); status != models.StatusPublishedWithNewerDraft {
		t.Errorf("status after rollback = %q, want published-with-newer-draft (head is v2)", status)
	}
}

func TestPublish_RejectsForeignVersion(t *testing.T) {
	st, docID, _ := publishFixture(t)
	ctx := tenantCtx(testTenant)

	// A version belonging to a different document.
	other, err := seedDoc(st, ctx, seedApp(t, st, ctx).ID, "other.md")
	if err != nil {
		t.Fatalf("create other doc: %v", err)
	}
	foreign, err := st.Versions.Save(ctx, other.ID, "x", 0)
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

// currentRelease loads the app's current release row.
func currentRelease(t *testing.T, st *stores.Stores, ctx context.Context, appID string) *models.Release {
	t.Helper()
	app, err := st.Apps.Get(ctx, appID)
	if err != nil || app.CurrentReleaseID == nil {
		t.Fatalf("app %s has no current release: %v", appID, err)
	}
	r, _, err := st.Releases.Get(ctx, appID, *app.CurrentReleaseID)
	if err != nil {
		t.Fatalf("current release: %v", err)
	}
	return r
}

// The publish sugar records what it was: a publish or unpublish release about
// the document, with the one-path diff counted.
func TestPublish_RecordsKindAndSubject(t *testing.T) {
	st, docID, versionID := publishFixture(t)
	ctx := tenantCtx(testTenant)

	doc, err := st.Publish(ctx, docID, versionID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	pub := currentRelease(t, st, ctx, doc.AppID)
	if pub.Kind != models.ReleaseKindPublish || pub.SubjectDocumentID == nil || *pub.SubjectDocumentID != docID || pub.Label != nil {
		t.Errorf("publish release = kind %q subject %v label %v; want publish about the document, unlabelled", pub.Kind, pub.SubjectDocumentID, pub.Label)
	}
	if pub.EntryCount != 1 || pub.Added != 1 || pub.Changed+pub.Removed+pub.Moved != 0 {
		t.Errorf("publish release counts = %+v, want one entry, one addition", pub)
	}
	_, entries, _ := st.Releases.Get(ctx, doc.AppID, pub.ID)
	if len(entries) != 1 || entries[0].VersionNumber != 1 {
		t.Errorf("publish release entries = %+v, want the page at version number 1", entries)
	}

	if _, err := st.Unpublish(ctx, docID); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	unpub := currentRelease(t, st, ctx, doc.AppID)
	if unpub.Kind != models.ReleaseKindUnpublish || unpub.SubjectDocumentID == nil || *unpub.SubjectDocumentID != docID {
		t.Errorf("unpublish release = kind %q subject %v; want unpublish about the document", unpub.Kind, unpub.SubjectDocumentID)
	}
	if unpub.EntryCount != 0 || unpub.Removed != 1 {
		t.Errorf("unpublish release counts = %+v, want no entries, one removal", unpub)
	}
	_, changes, _ := st.Releases.Changes(ctx, doc.AppID, unpub.ID)
	if len(changes) != 1 || changes[0].Change != models.ChangeRemoved || changes[0].DocumentID != docID || changes[0].VersionID != nil {
		t.Errorf("unpublish changes = %+v, want the one removal", changes)
	}
}
