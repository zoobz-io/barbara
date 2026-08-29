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
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// tagsFixture builds the aggregate over real Postgres (search is a stub — tag
// changes enqueue jobs and never call OpenSearch directly) and a bare document.
func tagsFixture(t *testing.T) (*stores.Stores, string) {
	t.Helper()
	db := pgDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM jobs")
		_, _ = db.Exec("DELETE FROM versions")
		_, _ = db.Exec("DELETE FROM documents")
		_ = db.Close()
	})
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	doc, err := seedDoc(st, tenantCtx(testTenant), seedApp(t, st, tenantCtx(testTenant)).ID, "taggable.md")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	return st, doc.ID
}

func TestTags_AddRemoveOnDraft_PostgresOnly(t *testing.T) {
	st, docID := tagsFixture(t)
	ctx := tenantCtx(testTenant)

	doc, err := st.AddTag(ctx, docID, "guide")
	if err != nil {
		t.Fatalf("add tag: %v", err)
	}
	if len(doc.Tags) != 1 || doc.Tags[0] != "guide" {
		t.Errorf("tags = %v, want [guide]", doc.Tags)
	}
	if doc.PublishedVersionID != nil {
		t.Errorf("draft gained a published pointer: %v", doc.PublishedVersionID)
	}

	// Adding the same tag again is a no-op.
	doc, err = st.AddTag(ctx, docID, "guide")
	if err != nil {
		t.Fatalf("add duplicate tag: %v", err)
	}
	if len(doc.Tags) != 1 {
		t.Errorf("duplicate add changed tags: %v", doc.Tags)
	}

	// Removing a tag it doesn't carry is a no-op; removing one it does clears it.
	if _, err = st.RemoveTag(ctx, docID, "absent"); err != nil {
		t.Fatalf("remove absent tag: %v", err)
	}
	doc, err = st.RemoveTag(ctx, docID, "guide")
	if err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	if len(doc.Tags) != 0 {
		t.Errorf("tags = %v, want empty", doc.Tags)
	}

	// A draft tag change never touches the outbox.
	claimed, err := st.Jobs.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 0 {
		t.Errorf("draft tag changes enqueued %d jobs, want 0", len(claimed))
	}
}

func TestTags_AddOnPublished_ReprojectsWithoutMovingPointer(t *testing.T) {
	st, docID := tagsFixture(t)
	ctx := tenantCtx(testTenant)

	v, err := st.Versions.Save(ctx, docID, "# content", 0)
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err = st.Publish(ctx, docID, v.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err = st.Jobs.ClaimPending(ctx, 10); err != nil { // drain the publish index job
		t.Fatalf("drain publish job: %v", err)
	}

	doc, err := st.AddTag(ctx, docID, "guide")
	if err != nil {
		t.Fatalf("add tag: %v", err)
	}

	// The published pointer is untouched — a tag change is metadata, not a publish.
	if doc.PublishedVersionID == nil || *doc.PublishedVersionID != v.ID {
		t.Errorf("published_version_id = %v, want %s (unchanged)", doc.PublishedVersionID, v.ID)
	}

	// A reprojection was enqueued carrying the new tag and the still-published
	// version's content.
	claimed, err := st.Jobs.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("enqueued %d jobs, want 1 reprojection", len(claimed))
	}
	job := claimed[0]
	if job.Operation != models.JobIndex || job.DocumentID != docID {
		t.Errorf("unexpected job: op=%s doc=%s", job.Operation, job.DocumentID)
	}
	var idx models.DocumentIndex
	if err := json.Unmarshal(job.Payload, &idx); err != nil {
		t.Fatalf("projection is not valid json: %v", err)
	}
	if idx.VersionID != v.ID || idx.Content != "# content" {
		t.Errorf("reprojection lost the published version: %+v", idx)
	}
	if len(idx.Tags) != 1 || idx.Tags[0] != "guide" {
		t.Errorf("reprojection tags = %v, want [guide]", idx.Tags)
	}
}

func TestTags_NoOpOnPublished_EnqueuesNothing(t *testing.T) {
	st, docID := tagsFixture(t)
	ctx := tenantCtx(testTenant)

	if _, err := st.AddTag(ctx, docID, "guide"); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	v, err := st.Versions.Save(ctx, docID, "# content", 0)
	if err != nil {
		t.Fatalf("save version: %v", err)
	}
	if _, err = st.Publish(ctx, docID, v.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err = st.Jobs.ClaimPending(ctx, 10); err != nil { // drain publish job
		t.Fatalf("drain: %v", err)
	}

	// Re-adding a tag the published doc already carries changes nothing, so no
	// reprojection is enqueued.
	if _, err = st.AddTag(ctx, docID, "guide"); err != nil {
		t.Fatalf("add duplicate tag: %v", err)
	}
	claimed, err := st.Jobs.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 0 {
		t.Errorf("no-op tag change enqueued %d jobs, want 0", len(claimed))
	}
}

func TestTags_ListByTag_TenantScoped(t *testing.T) {
	st, docID := tagsFixture(t)
	ctx := tenantCtx(testTenant)

	if _, err := st.AddTag(ctx, docID, "guide"); err != nil {
		t.Fatalf("tag d1: %v", err)
	}
	// A second doc without the tag (must be excluded), and one under another
	// tenant with the tag (must not leak across tenants).
	if _, err := seedDoc(st, ctx, seedApp(t, st, ctx).ID, "untagged.md"); err != nil {
		t.Fatalf("create untagged: %v", err)
	}
	foreign, err := seedDoc(st, tenantCtx(otherTenant), seedApp(t, st, tenantCtx(otherTenant)).ID, "foreign.md")
	if err != nil {
		t.Fatalf("create foreign: %v", err)
	}
	if _, err := st.AddTag(tenantCtx(otherTenant), foreign.ID, "guide"); err != nil {
		t.Fatalf("tag foreign: %v", err)
	}

	got, err := st.Documents.ListByTag(ctx, "guide", 50, 0)
	if err != nil {
		t.Fatalf("list by tag: %v", err)
	}
	if len(got) != 1 || got[0].ID != docID {
		t.Errorf("ListByTag returned %d docs, want just this tenant's tagged doc", len(got))
	}

	// No-tenant list is refused.
	if _, err := st.Documents.ListByTag(context.Background(), "guide", 50, 0); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("no-tenant ListByTag = %v, want ErrNoTenant", err)
	}
}
