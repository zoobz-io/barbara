package handlers

import "github.com/zoobz-io/rocco"

// All returns every admin (internal-platform) handler for registration in
// cmd/admin. The surface is deliberately small — cross-tenant search is the one
// capability seeded here; the platform grows from this.
func All() []rocco.Endpoint {
	return []rocco.Endpoint{
		SearchAll,
	}
}
