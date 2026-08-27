// Package auth resolves the identity and entitlement of an incoming request.
//
// Barbara has no user table. Identity and entitlement belong to janus, reached
// over the aegis mesh; services authenticate to each other with mesh CA client
// certificates (docs/plans/001-domain.md, "Auth"). Until that integration
// lands, a stub resolver stands in so surface work is not blocked — see
// [StubAuthenticator]. The real janus/aegis resolver is a drop-in that
// satisfies [Authenticator]; nothing else in the package changes.
//
// Every resolved request produces a [Principal]: the tenant it operates on, the
// acting user, and the entitlement (roles/scopes) granted for that tenant.
package auth

import "github.com/zoobz-io/rocco"

// Compile-time assertion that Principal satisfies rocco's Identity contract.
var _ rocco.Identity = (*Principal)(nil)

// Principal is barbara's resolved request identity. It carries the tenant the
// request operates on, the acting user, and the entitlement granted for that
// tenant, and implements rocco.Identity so handlers read it off req.Identity.
//
// Whatever resolves the request — the dev stub today, janus/aegis later —
// produces a Principal; the shape is stable across both.
type Principal struct {
	userID   string
	tenantID string
	email    string
	roles    []string
	scopes   []string
}

// NewPrincipal builds a Principal for the acting user on the given tenant with
// the supplied entitlement. roles and scopes are defensively copied so the
// caller cannot mutate the identity after construction.
func NewPrincipal(userID, tenantID, email string, roles, scopes []string) *Principal {
	return &Principal{
		userID:   userID,
		tenantID: tenantID,
		email:    email,
		roles:    append([]string(nil), roles...),
		scopes:   append([]string(nil), scopes...),
	}
}

// ID returns the acting user's identifier.
func (p *Principal) ID() string { return p.userID }

// TenantID returns the tenant the request operates on. Every query is
// tenant-scoped, so this is the value stores filter by.
func (p *Principal) TenantID() string { return p.tenantID }

// Email returns the acting user's email, or empty if the resolver does not
// supply one (janus session validation does not).
func (p *Principal) Email() string { return p.email }

// Scopes returns the tenant-scoped permissions granted to this identity.
func (p *Principal) Scopes() []string { return p.scopes }

// Roles returns the tenant-scoped roles assigned to this identity.
func (p *Principal) Roles() []string { return p.roles }

// HasScope reports whether the identity holds the given scope.
func (p *Principal) HasScope(scope string) bool {
	for _, s := range p.scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasRole reports whether the identity holds the given role.
func (p *Principal) HasRole(role string) bool {
	for _, r := range p.roles {
		if r == role {
			return true
		}
	}
	return false
}

// Stats returns nil — rate-limiting metrics are not tracked on the identity.
func (p *Principal) Stats() map[string]int { return nil }
