package auth

import (
	"context"
	"net/http"

	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"
)

// NewExtractor adapts an Authenticator to the identity-extraction function
// rocco's engine expects. rocco calls it for every handler marked
// WithAuthentication(), stores the result on req.Identity, and rejects the
// request with 401 on error.
func NewExtractor(a Authenticator) func(context.Context, *http.Request) (rocco.Identity, error) {
	return func(ctx context.Context, r *http.Request) (rocco.Identity, error) {
		return a.Authenticate(ctx, r)
	}
}

// Wire installs the request auth for a surface: it registers the resolver in
// the sum registry under the Authenticator contract, and sets it as the
// engine's identity extractor so WithAuthentication() handlers resolve through
// it. Both binaries call this once, before Freeze.
//
// Registering under the contract is what makes the janus/aegis resolver a
// drop-in: swap the argument here (or the registration) and no handler changes.
func Wire(k sum.Key, engine *rocco.Engine, a Authenticator) {
	sum.Register[Authenticator](k, a)
	engine.WithAuthenticator(NewExtractor(a))
}
