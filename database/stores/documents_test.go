//go:build testing

package stores

import "testing"

// Create inserts every non-PK column for the request's tenant and returns the
// generated row; the id is DB-generated, so it is omitted from the column list.
func TestDocuments_Create_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Documents.Create(tenantCtx(), "a.md")

	q := lastQuery(t, capture)
	wantSQL(t, q, `INSERT INTO "documents"`, `"tenant_id"`, `"key"`, `"tags"`, `RETURNING`)
	wantArg(t, q, testTenant)
	wantArg(t, q, "a.md")
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

// Rename updates key + updated_at, scoped to the tenant, and returns the row.
func TestDocuments_Rename_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Documents.Rename(tenantCtx(), "d-1", "b.md")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "documents" SET`,
		`"key" = ?`,
		`"updated_at" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`RETURNING`,
	)
	wantArg(t, q, "b.md")
	wantArg(t, q, testTenant)
}

// Delete refuses a published document at the query level: the WHERE carries
// published_version_id IS NULL alongside the tenant scope, so a published row
// never matches.
func TestDocuments_Delete_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_ = st.Documents.Delete(tenantCtx(), "d-1")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`DELETE FROM "documents"`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`"published_version_id" IS NULL`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}
