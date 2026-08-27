package auth

import (
	"context"

	"github.com/zoobz-io/rocco"
)

// principalContextKey is barbara's private context key for the resolved
// identity. rocco stores the identity under its own unexported key (readable
// only as req.Identity in a handler); this key is what lets barbara carry the
// identity further down — into tenant-scoped stores that see only a context.
type principalContextKey struct{}

// WithPrincipal returns a context carrying the resolved identity. A handler
// bridges req.Identity into the context with this before calling tenant-scoped
// stores, so the store can read the tenant from the context alone.
func WithPrincipal(ctx context.Context, id rocco.Identity) context.Context {
	return context.WithValue(ctx, principalContextKey{}, id)
}

// PrincipalFromContext returns the identity carried by the context, if any.
func PrincipalFromContext(ctx context.Context) (rocco.Identity, bool) {
	id, ok := ctx.Value(principalContextKey{}).(rocco.Identity)
	return id, ok
}

// TenantFromContext returns the tenant the request operates on, or empty if no
// identity is carried. Tenant-scoped stores use this to scope every query; a
// store that refuses to run without a tenant treats empty as unauthenticated.
func TenantFromContext(ctx context.Context) string {
	if id, ok := PrincipalFromContext(ctx); ok {
		return id.TenantID()
	}
	return ""
}

// UserFromContext returns the acting user, or empty if no identity is carried.
// Authoring writes stamp created_by/updated_by with it.
func UserFromContext(ctx context.Context) string {
	if id, ok := PrincipalFromContext(ctx); ok {
		return id.ID()
	}
	return ""
}
