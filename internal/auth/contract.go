package auth

import (
	"context"
	"net/http"

	"github.com/zoobz-io/rocco"
)

// Authenticator resolves the identity and entitlement of an incoming request:
// given the request, it returns the tenant it operates on, the acting user, and
// the entitlement granted for that tenant, as a rocco.Identity (concretely a
// [Principal]). A non-nil error means the request is unauthenticated or not
// authorized for the requested tenant — rocco turns that into a 401.
//
// This is the seam the real janus/aegis integration implements. It is
// registered in the sum registry (see [Wire]) so swapping the dev stub for the
// mesh resolver is a registration change, not a call-site change. The resolver
// must:
//
//   - authenticate the caller (mesh client cert for services; a janus session
//     token for users);
//   - determine which tenant the request operates on and verify the caller is
//     authorized for it (argus selects it with an X-Tenant-ID header);
//   - return a Principal carrying tenant, acting user, and the tenant-scoped
//     roles/scopes janus reports.
type Authenticator interface {
	// Authenticate resolves the request into an identity, or errors if the
	// request cannot be authenticated or is not authorized for its tenant.
	Authenticate(ctx context.Context, r *http.Request) (rocco.Identity, error)
}
