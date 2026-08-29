//go:build testing

package stores

import (
	"context"
	"errors"
	"testing"

	"github.com/lib/pq"

	"github.com/zoobz-io/barbara/internal/auth"
)

// Create at the app root materializes the key from the name, checks no sibling
// collection holds the name, and inserts with the tree columns set.
func TestDocuments_Create_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())    // requireParentScope: apps.Get
	cfg.PushRowData(countRow(0)) // siblingCollectionExists: no collection sibling
	cfg.PushRowData(docRow(nil)) // INSERT ... RETURNING

	_, err := st.Documents.Create(tenantCtx(), testApp, nil, "a.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	sib := queryAt(t, capture, 1)
	wantSQL(t, sib, `SELECT COUNT(*) FROM "collections"`, `"app_id" = ?`, `"name" = ?`, `"parent_id" IS NULL`)
	wantArg(t, sib, "a.md")

	ins := queryAt(t, capture, 2)
	wantSQL(t, ins, `INSERT INTO "documents"`, `"app_id"`, `"collection_id"`, `"name"`, `"key"`, `"tags"`, `RETURNING`)
	wantArg(t, ins, testTenant)
	wantArg(t, ins, "a.md") // both name and key (root key == name)
}

// Get is scoped to the request's tenant: id AND tenant_id.
func TestDocuments_Get_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Documents.Get(tenantCtx(), "d-1")

	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "documents"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

// List is tenant-scoped, oldest first, and paginated with LIMIT/OFFSET.
func TestDocuments_List_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Documents.List(tenantCtx(), 10, 5)

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "documents"`,
		`"tenant_id" = ?`,
		`ORDER BY "created_at" ASC`,
		`LIMIT 10`,
		`OFFSET 5`,
	)
}

// ListByTag adds a Postgres array-containment filter (tags @> ARRAY[tag]) on top
// of the tenant scope, with the tag bound as a single-element array.
func TestDocuments_ListByTag_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Documents.ListByTag(tenantCtx(), "guide", 10, 5)

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`"tenant_id" = ?`,
		`"tags" @> ?`,
		`ORDER BY "created_at" ASC`,
		`LIMIT 10`,
		`OFFSET 5`,
	)
	// The tag is bound as a pq array literal, not a bare string.
	wantArg(t, q, `{"guide"}`)
}

// Move locks the document, then updates collection_id, name, and the rewritten
// key, tenant-scoped, returning the row.
func TestDocuments_Move_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())    // requireParentScope: apps.Get (new parent = root)
	cfg.PushRowData(docRow(nil)) // lock (FOR UPDATE)
	cfg.PushRowData(countRow(0)) // siblingCollectionExists
	cfg.PushRowData(docRow(nil)) // UPDATE ... RETURNING

	_, err := st.Documents.Move(tenantCtx(), testApp, "d-1", nil, "b.md")
	if err != nil {
		t.Fatalf("move: %v", err)
	}

	lock := queryAt(t, capture, 1)
	wantSQL(t, lock, `FROM "documents"`, `"id" = ?`, `"app_id" = ?`, `FOR UPDATE`)

	upd := queryAt(t, capture, 3)
	wantSQL(t, upd,
		`UPDATE "documents" SET`,
		`"collection_id" = ?`,
		`"name" = ?`,
		`"key" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`RETURNING`,
	)
	wantArg(t, upd, "b.md")
	wantArg(t, upd, testTenant)
}

// Delete loads the document, and — for an unplaced one not in any release —
// hard-deletes it, tenant-scoped.
func TestDocuments_Delete_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil)) // Get: a draft, unplaced document (app_id nil)

	_ = st.Documents.Delete(tenantCtx(), "d-1")

	del := lastQuery(t, capture)
	wantSQL(t, del, `DELETE FROM "documents"`, `"id" = ?`, `"tenant_id" = ?`)
	notSQL(t, del, "published_version_id")
	wantArg(t, del, "d-1")
	wantArg(t, del, testTenant)
}

// A sibling collection sharing the name blocks a create, before any INSERT.
func TestDocuments_Create_SiblingCollectionBlocks(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())    // scope ok
	cfg.PushRowData(countRow(1)) // a collection sibling holds the name

	_, err := st.Documents.Create(tenantCtx(), testApp, nil, "guides")
	if !errors.Is(err, ErrCollectionNameTaken) {
		t.Fatalf("create colliding with a sibling collection = %v, want ErrCollectionNameTaken", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "documents"`)
	}
}

// A sibling collection at the destination blocks a move, before the UPDATE.
func TestDocuments_Move_SiblingCollectionBlocks(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())    // requireParentScope
	cfg.PushRowData(docRow(nil)) // lock
	cfg.PushRowData(countRow(1)) // sibling collection at the destination

	_, err := st.Documents.Move(tenantCtx(), testApp, "d-1", nil, "guides")
	if !errors.Is(err, ErrCollectionNameTaken) {
		t.Fatalf("move colliding with a sibling collection = %v, want ErrCollectionNameTaken", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `UPDATE "documents" SET`)
	}
}

// When the hard delete hits the release_entries RESTRICT foreign key, the
// document is soft-deleted instead: deleted_at is set, the row survives.
func TestDocuments_Delete_SoftOnForeignKey(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil))              // Get: a draft, unplaced doc
	cfg.PushExecErr(&pq.Error{Code: "23503"}) // hard DELETE hits the release FK
	cfg.PushRowData(docRow(nil))              // soft UPDATE ... RETURNING

	if err := st.Documents.Delete(tenantCtx(), "d-1"); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	soft := lastQuery(t, capture)
	wantSQL(t, soft, `UPDATE "documents" SET`, `"deleted_at" = ?`, `"id" = ?`, `"tenant_id" = ?`)
}

// Every tree mutation requires a tenant in context.
func TestDocuments_RequiresTenant(t *testing.T) {
	st, _ := newQueryTest(t)
	ctx := context.Background()
	if _, err := st.Documents.Create(ctx, testApp, nil, "x"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Create tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Documents.Move(ctx, testApp, "d-1", nil, "x"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Move tenantless = %v, want ErrNoTenant", err)
	}
	if err := st.Documents.Delete(ctx, "d-1"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Delete tenantless = %v, want ErrNoTenant", err)
	}
}

// A document carried by the app's current release is refused (unpublish first),
// before any DELETE.
func TestDocuments_Delete_RefusesInCurrentRelease(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))             // Get: a placed document
	cfg.PushRowData(appRowWithRelease("r-1"))    // its app has a current release
	cfg.PushRowData(countRow(1))                 // ...which carries the document

	if err := st.Documents.Delete(tenantCtx(), "d-1"); !errors.Is(err, ErrDocumentPublished) {
		t.Fatalf("delete of a live doc = %v, want ErrDocumentPublished", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `DELETE FROM "documents"`)
	}
}

// GetWithHead issues the tenant-scoped document read, then the head-version
// lookup — the doc plus its latest version in one method.
func TestDocuments_GetWithHead_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(nil))  // Documents.Get succeeds
	cfg.PushRowData(versionRow())  // Versions.Head returns the head
	_, _ = st.Documents.GetWithHead(tenantCtx(), "d-1")

	get := queryAt(t, capture, 0)
	wantSQL(t, get, `FROM "documents"`, `"id" = ?`, `"tenant_id" = ?`)

	head := queryAt(t, capture, 1)
	wantSQL(t, head,
		`FROM "versions"`,
		`"document_id" = ?`,
		`ORDER BY "version_number" DESC`,
		`LIMIT 1`,
	)
}
