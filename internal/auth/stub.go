package auth

import (
	"context"
	"net/http"

	"github.com/zoobz-io/rocco"
)

// Dev defaults for the stub resolver. The tenant and user are valid UUIDs —
// barbara's tenant_id/created_by columns are UUID — but deliberately all-ones/
// all-twos so a stubbed identity is never mistaken for a real one in logs or
// data.
const (
	// DefaultTenantID is the tenant the stub resolves to when no X-Tenant-ID
	// header overrides it.
	DefaultTenantID = "11111111-1111-1111-1111-111111111111"
	// DefaultUserID is the acting user the stub resolves to.
	DefaultUserID = "22222222-2222-2222-2222-222222222222"
	// DefaultEmail is the acting user's email under the stub.
	DefaultEmail = "dev@barbara.local"
	// RoleAdmin is the dev role granted by the stub — broad enough that local
	// authoring handlers pass entitlement checks.
	RoleAdmin = "admin"
)

// Compile-time assertion that StubAuthenticator satisfies the contract.
var _ Authenticator = (*StubAuthenticator)(nil)

// StubAuthenticator is the local-dev and test resolver: it authenticates
// nothing and resolves every request to a fixed, injected identity. It stands
// in for janus/aegis so surface work proceeds before the mesh integration
// lands.
//
// It honors an X-Tenant-ID header override so multi-tenant behaviour can be
// exercised locally and in tests — mirroring how the real resolver selects a
// tenant — while falling back to the injected default tenant otherwise.
//
// It is not an authenticator in any security sense and must never be registered
// in a deployed binary.
type StubAuthenticator struct {
	userID   string
	tenantID string
	email    string
	roles    []string
	scopes   []string
}

// NewStub builds a stub resolver that resolves every request to the given
// identity. An empty tenantID/userID falls back to the dev defaults.
func NewStub(userID, tenantID, email string, roles, scopes []string) *StubAuthenticator {
	if userID == "" {
		userID = DefaultUserID
	}
	if tenantID == "" {
		tenantID = DefaultTenantID
	}
	return &StubAuthenticator{
		userID:   userID,
		tenantID: tenantID,
		email:    email,
		roles:    append([]string(nil), roles...),
		scopes:   append([]string(nil), scopes...),
	}
}

// DefaultStub returns the stub resolver with the dev defaults: the dev user on
// the dev tenant, holding the admin role and every authoring scope — broad
// enough that local callers pass all entitlement gates.
func DefaultStub() *StubAuthenticator {
	return NewStub(DefaultUserID, DefaultTenantID, DefaultEmail, []string{RoleAdmin}, AuthoringScopes())
}

// Authenticate resolves the request to the stub's fixed identity. If the
// request carries an X-Tenant-ID header, that tenant is used instead of the
// default — so tests and local callers can act as different tenants. It never
// returns an error: the stub authenticates nothing.
func (s *StubAuthenticator) Authenticate(_ context.Context, r *http.Request) (rocco.Identity, error) {
	tenantID := s.tenantID
	if r != nil {
		if override := r.Header.Get("X-Tenant-ID"); override != "" {
			tenantID = override
		}
	}
	return NewPrincipal(s.userID, tenantID, s.email, s.roles, s.scopes), nil
}
