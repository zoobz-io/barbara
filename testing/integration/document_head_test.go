//go:build testing

package integration

import (
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// TestDocumentHead_Integration drives the single-call editing read (#48) against
// real Postgres: opening an empty document, then one carrying versions, and
// confirming the head version and its content come back with the document.
func TestDocumentHead_Integration(t *testing.T) {
	db := pgDB(t)
	const tenant = testTenant
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM jobs WHERE tenant_id = $1", tenant)
		_, _ = db.Exec("DELETE FROM versions WHERE tenant_id = $1", tenant)
		_, _ = db.Exec("DELETE FROM documents WHERE tenant_id = $1", tenant)
		_ = db.Close()
	})
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	ctx := tenantCtx(tenant)

	doc, err := st.Documents.Create(ctx, "editor/doc.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Empty document: a null head and no published pointer — not a 404.
	dh, err := st.Documents.GetWithHead(ctx, doc.ID)
	if err != nil {
		t.Fatalf("get-with-head (empty): %v", err)
	}
	if dh.Head != nil {
		t.Errorf("empty document has a head version: %+v", dh.Head)
	}
	if dh.Document.PublishedVersionID != nil {
		t.Errorf("empty document is published: %v", dh.Document.PublishedVersionID)
	}

	// Two versions: the head is the latest, and its content comes back in the
	// single read.
	if _, err := st.Versions.Save(ctx, doc.ID, "# v1"); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	v2, err := st.Versions.Save(ctx, doc.ID, "# v2")
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	dh, err = st.Documents.GetWithHead(ctx, doc.ID)
	if err != nil {
		t.Fatalf("get-with-head (v2): %v", err)
	}
	if dh.Head == nil || dh.Head.ID != v2.ID || dh.Head.Content != "# v2" {
		t.Fatalf("head = %+v, want v2 with its content", dh.Head)
	}

	// Publishing sets the pointer; the head still reads back as the latest version.
	if _, err := st.Publish(ctx, doc.ID, v2.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	dh, _ = st.Documents.GetWithHead(ctx, doc.ID)
	if dh.Document.PublishedVersionID == nil || *dh.Document.PublishedVersionID != v2.ID {
		t.Errorf("published pointer = %v, want %s", dh.Document.PublishedVersionID, v2.ID)
	}
	if dh.Head == nil || dh.Head.ID != v2.ID {
		t.Errorf("head after publish = %+v, want v2", dh.Head)
	}
}
