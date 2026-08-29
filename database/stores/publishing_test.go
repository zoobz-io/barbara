//go:build testing

package stores

import "testing"

// Publish first loads the version to be published, scoped to the request's
// tenant (id AND tenant_id) — a version from another tenant is never visible.
func TestPublish_ValidatesVersionInTenant(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Publish(tenantCtx(), "d-1", "v-1")

	// The tenant-scoped version lookup is the first query; the mock returns no
	// row, so the pointer move never runs and this is the captured query.
	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "versions"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "v-1")
	wantArg(t, q, testTenant)
}

// Unpublish clears the published pointer (published_version_id = NULL) and bumps
// updated_at, scoped to id + tenant, returning the updated row.
func TestUnpublish_ClearsPointer(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Unpublish(tenantCtx(), "d-1")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "documents" SET`,
		`"published_version_id" = ?`,
		`"updated_at" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`RETURNING`,
	)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}
