// Package handlers defines the site-facing (api) HTTP endpoints, served to mesh
// services from OpenSearch exclusively. Handlers are thin: resolve the
// contract, bridge the request identity into the context so the search store is
// tenant-scoped, call the contract, and transform the result.
package handlers

import (
	"errors"
	"strconv"

	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

// GetPublishedDocument returns a published document by key. The key is
// path-like (it carries slashes), so it is passed as a query parameter rather
// than a path segment.
var GetPublishedDocument = rocco.GET("/documents/lookup",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentResponse, error) {
		key := req.Params.Query["key"]
		if key == "" {
			return wire.PublishedDocumentResponse{}, rocco.ErrBadRequest.WithMessage("key query parameter required")
		}
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := reads.GetPublishedByKey(ctx, key)
		if err != nil {
			return wire.PublishedDocumentResponse{}, mapError(err)
		}
		return transformers.IndexToResponse(doc), nil
	}).WithQueryParams("key").
	WithSummary("Get a published document by key").
	WithTags("Documents").
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// EnumerateDocuments lists published documents for the tenant, optionally
// filtered by tag. Enumeration is required because new files create new pages.
var EnumerateDocuments = rocco.GET("/documents",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := pagination(req)
		docs, total, err := reads.Enumerate(ctx, req.Params.Query["tag"], limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, mapError(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithQueryParams("tag", "limit", "offset").
	WithSummary("Enumerate published documents").
	WithTags("Documents").
	WithErrors(rocco.ErrUnauthorized).
	WithAuthentication()

// SearchDocuments runs a full-text search over published content for the
// tenant.
var SearchDocuments = rocco.GET("/search",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		query := req.Params.Query["q"]
		if query == "" {
			return wire.PublishedDocumentListResponse{}, rocco.ErrBadRequest.WithMessage("q query parameter required")
		}
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := pagination(req)
		docs, total, err := reads.Search(ctx, query, limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, mapError(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithQueryParams("q", "limit", "offset").
	WithSummary("Full-text search over published documents").
	WithTags("Documents").
	WithErrors(rocco.ErrBadRequest, rocco.ErrUnauthorized).
	WithAuthentication()

// pagination reads limit/offset from the query, applying defaults.
func pagination(req *rocco.Request[rocco.NoBody]) (limit, offset int) {
	limit, offset = 50, 0
	if v := req.Params.Query["limit"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := req.Params.Query["offset"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

// mapError translates store errors into HTTP responses.
func mapError(err error) error {
	switch {
	case errors.Is(err, stores.ErrNotFound):
		return rocco.ErrNotFound.WithMessage("document not found")
	case errors.Is(err, auth.ErrNoTenant):
		return rocco.ErrUnauthorized.WithMessage("request has no tenant")
	default:
		return err
	}
}
