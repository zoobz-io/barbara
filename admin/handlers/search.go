// Package handlers defines the admin (internal-platform) HTTP endpoints. The
// admin surface is tenant-agnostic; every endpoint is gated behind an admin
// entitlement, distinct from the tenant scopes the public API uses.
package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/admin/contracts"
	"github.com/zoobz-io/barbara/admin/transformers"
	"github.com/zoobz-io/barbara/admin/wire"
	dbtransformers "github.com/zoobz-io/barbara/database/transformers"
	"github.com/zoobz-io/barbara/internal/auth"
)

// SearchAll runs a cross-tenant full-text search over published content. It is
// admin-only: gated behind the admin role, so a tenant user (who never holds it)
// cannot reach across tenants.
var SearchAll = rocco.GET("/search",
	func(req *rocco.Request[rocco.NoBody]) (wire.SearchResultsResponse, error) {
		query := req.Params.Query["q"]
		if query == "" {
			return wire.SearchResultsResponse{}, rocco.ErrBadRequest.WithMessage("q query parameter required")
		}
		search := sum.MustUse[contracts.Search](req.Context)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		results, total, err := search.SearchAll(req.Context, query, limit, offset)
		if err != nil {
			return wire.SearchResultsResponse{}, err
		}
		return transformers.SearchResultsToResponse(results, total, limit, offset), nil
	}).WithQueryParams("q", "limit", "offset").
	WithSummary("Search published documents across all tenants").
	WithTags("Admin").
	WithErrors(rocco.ErrBadRequest, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithRoles(auth.RoleAdmin).
	WithAuthentication()
