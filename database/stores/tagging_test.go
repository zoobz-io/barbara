//go:build testing

package stores

import "testing"

// AddTag locks the document row FOR UPDATE, scoped to id + tenant, so concurrent
// tag changes serialize on that row rather than losing each other's updates.
func TestAddTag_LocksDocumentForUpdate(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.AddTag(tenantCtx(), "d-1", "guide")

	// The lock select is the first query; the mock returns no row, so the tag
	// write never runs and this is the captured query.
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

// RemoveTag takes the same tenant-scoped FOR UPDATE lock as AddTag.
func TestRemoveTag_LocksDocumentForUpdate(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.RemoveTag(tenantCtx(), "d-1", "guide")

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
