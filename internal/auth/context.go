package auth

import (
	"context"
	"errors"

	"github.com/zoobz-io/rocco"
)

// ErrNoTenant is returned when a request reaches a tenant-scoped store without a
// tenant — every query is tenant-scoped, so the store refuses to run.
var ErrNoTenant = errors.New("no tenant in context")

// ErrNoUser is returned when a write reaches a store without an acting user —
// authoring writes record who made them.
var ErrNoUser = errors.New("no acting user in context")

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

// RequireTenant returns the request's tenant, or ErrNoTenant if none is carried.
// Tenant-scoped stores call this instead of extracting the tenant themselves.
func RequireTenant(ctx context.Context) (string, error) {
	if id := TenantFromContext(ctx); id != "" {
		return id, nil
	}
	return "", ErrNoTenant
}

// RequireUser returns the acting user, or ErrNoUser if none is carried. Stores
// that stamp created_by call this.
func RequireUser(ctx context.Context) (string, error) {
	if id := UserFromContext(ctx); id != "" {
		return id, nil
	}
	return "", ErrNoUser
}
