// Package handlers defines the public-API HTTP endpoints: the site-facing
// published reads (served from OpenSearch under /published/*) and the
// tenant-scoped authoring surface — documents, versions, tags, publishing,
// assets. Handlers are thin: resolve the
// contract, bridge the request identity into the context so the store is
// tenant-scoped, call the contract, and transform the result.
package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	dbtransformers "github.com/zoobz-io/barbara/database/transformers"
	"github.com/zoobz-io/barbara/internal/auth"
)

// GetPublishedDocument returns a published document by key. The key is
// path-like (it carries slashes), so it is passed as a query parameter rather
// than a path segment.
var GetPublishedDocument = rocco.GET("/published/lookup",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentResponse, error) {
		key := req.Params.Query["key"]
		if key == "" {
			return wire.PublishedDocumentResponse{}, rocco.ErrBadRequest.WithMessage("key query parameter required")
		}
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := reads.GetPublishedByKey(ctx, key)
		if err != nil {
			return wire.PublishedDocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexToResponse(doc), nil
	}).WithQueryParams("key").
	WithSummary("Get a published document by key").
	WithTags("Documents").
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// EnumerateDocuments lists published documents for the tenant, optionally
// filtered by tag. Enumeration is required because new files create new pages.
var EnumerateDocuments = rocco.GET("/published/documents",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		docs, total, err := reads.Enumerate(ctx, req.Params.Query["tag"], limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithQueryParams("tag", "limit", "offset").
	WithSummary("Enumerate published documents").
	WithTags("Documents").
	WithErrors(rocco.ErrUnauthorized).
	WithAuthentication()

// SearchDocuments runs a full-text search over published content for the
// tenant.
var SearchDocuments = rocco.GET("/published/search",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		query := req.Params.Query["q"]
		if query == "" {
			return wire.PublishedDocumentListResponse{}, rocco.ErrBadRequest.WithMessage("q query parameter required")
		}
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		docs, total, err := reads.Search(ctx, query, limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithQueryParams("q", "limit", "offset").
	WithSummary("Full-text search over published documents").
	WithTags("Documents").
	WithErrors(rocco.ErrBadRequest, rocco.ErrUnauthorized).
	WithAuthentication()
