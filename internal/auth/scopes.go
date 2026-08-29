package auth

// Authoring scopes gate the tenant-facing document lifecycle on the public API.
// They live here, in one place, so handlers reference constants rather than
// scattering string literals — and so the mesh resolver and the dev stub issue
// the same names. Published reads (lookup/enumerate/search) are the site's
// render path and stay behind plain tenant authentication, ungated.
const (
	// ScopeDocumentsRead covers authoring reads: documents, versions, and asset
	// reads.
	ScopeDocumentsRead = "documents:read"
	// ScopeDocumentsWrite covers create/rename/delete, saving a version, tag
	// changes, and asset writes.
	ScopeDocumentsWrite = "documents:write"
	// ScopeDocumentsPublish covers publish, unpublish, and rollback.
	ScopeDocumentsPublish = "documents:publish"
)

// AuthoringScopes is every authoring scope. The dev stub issues them all so
// local and test identities pass every gate; denial is exercised with a stub
// that omits a scope.
func AuthoringScopes() []string {
	return []string{ScopeDocumentsRead, ScopeDocumentsWrite, ScopeDocumentsPublish}
}
