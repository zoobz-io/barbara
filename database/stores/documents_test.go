//go:build testing

package stores

import (
	"errors"
	"testing"
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

// ListPublishedAfter is the reindex keyset page: published-only, id > afterID,
// ordered by id — and deliberately NOT tenant-scoped (it runs cross-tenant,
// outside any request context).
func TestDocuments_ListPublishedAfter_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Documents.ListPublishedAfter(tenantCtx(), zeroUUID, 100)

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "documents"`,
		`"published_version_id" IS NOT NULL`,
		`"id" > ?`,
		`ORDER BY "id" ASC`,
		`LIMIT 100`,
	)
	notSQL(t, q, "tenant_id")
	wantArg(t, q, zeroUUID)
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

// Delete loads the document, and — for an unpublished, unplaced one — hard-deletes
// it, tenant-scoped. The published-pointer guard now happens in Go against the
// loaded row, not in the DELETE's WHERE.
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

// A document still holding a published pointer is refused (unpublish first),
// before any DELETE.
func TestDocuments_Delete_RefusesPublished(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow("v-9")) // Get: published_version_id set

	if err := st.Documents.Delete(tenantCtx(), "d-1"); !errors.Is(err, ErrDocumentPublished) {
		t.Fatalf("delete of a published doc = %v, want ErrDocumentPublished", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `DELETE FROM "documents"`)
	}
}

// GetWithHead issues the tenant-scoped document read, then the head-version
// lookup — the doc plus its latest version in one method.
func TestDocuments_GetWithHead_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow("v-1")) // Documents.Get succeeds (published doc)
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
