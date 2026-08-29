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
	_, _ = st.Versions.Save(tenantCtx(), "d-1", "hello")

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
