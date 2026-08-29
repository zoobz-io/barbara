// Package contracts defines the interfaces the admin (internal-platform) surface
// depends on. Admin is tenant-agnostic — its identities carry platform
// entitlements distinct from tenant users — so its contracts are cross-tenant
// and gated behind an admin entitlement at the handler.
package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Search is the admin cross-tenant view over the published-document index. It is
// deliberately NOT tenant-scoped — seeing across tenants is the point of the
// admin surface — which is why it was kept off the site-facing reads contract.
type Search interface {
	// SearchAll runs a full-text search over published content across every tenant.
	SearchAll(ctx context.Context, query string, limit, offset int) ([]models.DocumentIndex, int64, error)
}
