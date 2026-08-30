//go:build testing

package stores

import "testing"

// Get is scoped to the request's tenant: id AND tenant_id.
func TestVersions_Get_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Versions.Get(tenantCtx(), "v-1")

	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "versions"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "v-1")
	wantArg(t, q, testTenant)
}

// GetByID is the reindex read: a bare primary-key lookup, deliberately NOT
// tenant-scoped (it runs cross-tenant, outside any request context).
func TestVersions_GetByID_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Versions.GetByID(tenantCtx(), "v-1")

	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "versions"`, `"id" = ?`)
	notSQL(t, q, "tenant_id")
	wantArg(t, q, "v-1")
}

// List is scoped to the tenant and the document, newest version first, paginated.
func TestVersions_List_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Versions.List(tenantCtx(), "d-1", 10, 5)

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "versions"`,
		`"document_id" = ?`,
		`"tenant_id" = ?`,
		`ORDER BY "version_number" DESC`,
		`LIMIT 10`,
		`OFFSET 5`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

// Save first locks the parent document row FOR UPDATE, scoped to id + tenant, so
// concurrent saves for the same document serialize on that row.
func TestVersions_Save_LocksParentForUpdate(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Versions.Save(tenantCtx(), "d-1", "hello", 0)

	// The parent-lock select is the first (and, since the mock returns no row,
	// the only) query the method issues.
	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "documents"`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`FOR UPDATE`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

// Save's full query sequence, driven to completion by feeding the lock row and
// the count: after the FOR UPDATE lock it counts the document's existing
// versions (scoped to document + tenant) and inserts the next version, numbering
// it count+1 and stamping the acting user.
func TestVersions_Save_CountsThenInsertsNextNumber(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp))  // parent lock select
	cfg.PushRowData(countRow(2))  // existing version count (head)
	cfg.PushRowData(versionRow()) // INSERT ... RETURNING
	// base_version must equal the head (2) so the save proceeds to the insert.
	_, _ = st.Versions.Save(tenantCtx(), "d-1", "hello", 2)

	// Query 1: count the document's versions, tenant-scoped.
	count := queryAt(t, capture, 1)
	wantSQL(t, count, `SELECT COUNT(*) FROM "versions"`, `"document_id" = ?`, `"tenant_id" = ?`)
	wantArg(t, count, "d-1")
	wantArg(t, count, testTenant)

	// Query 2: insert the next version — content, acting user, and version_number
	// = count + 1 (3, here).
	ins := queryAt(t, capture, 2)
	wantSQL(t, ins,
		`INSERT INTO "versions"`,
		`"content"`,
		`"created_by"`,
		`"document_id"`,
		`"version_number"`,
		`RETURNING`,
	)
	wantArg(t, ins, "hello")   // content
	wantArg(t, ins, testUser)  // created_by
	wantArg(t, ins, 3)         // version_number = existing count (2) + 1
}

// Head loads the document's latest version — tenant + document scoped, newest
// first, one row.
func TestVersions_Head_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Versions.Head(tenantCtx(), "d-1")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "versions"`,
		`"document_id" = ?`,
		`"tenant_id" = ?`,
		`ORDER BY "version_number" DESC`,
		`LIMIT 1`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}

