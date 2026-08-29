// Package handlers defines the public-API HTTP endpoints: the site-facing
// published reads (served from OpenSearch under /published/apps/{app_id}/*)
// and the tenant-scoped authoring surface — documents, versions, tags,
// publishing, assets. Handlers are thin: resolve the contract, bridge the
// request identity into the context so the store is tenant-scoped, call the
// contract, and transform the result.
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

// GetPublishedDocument returns an app's published document by key. The key is
// path-like (it carries slashes), so it is passed as a query parameter rather
// than a path segment.
var GetPublishedDocument = rocco.GET("/published/apps/{app_id}/lookup",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentResponse, error) {
		key := req.Params.Query["key"]
		if key == "" {
			return wire.PublishedDocumentResponse{}, rocco.ErrBadRequest.WithMessage("key query parameter required")
		}
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := reads.GetPublishedByKey(ctx, req.Params.Path["app_id"], key)
		if err != nil {
			return wire.PublishedDocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexToResponse(doc), nil
	}).WithPathParams("app_id").
	WithQueryParams("key").
	WithSummary("Get a published document by key").
	WithTags("Published").
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// EnumerateDocuments lists an app's published documents, optionally filtered
// by tag. Enumeration is required because new files create new pages.
var EnumerateDocuments = rocco.GET("/published/apps/{app_id}/documents",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		docs, total, err := reads.Enumerate(ctx, req.Params.Path["app_id"], req.Params.Query["tag"], limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithPathParams("app_id").
	WithQueryParams("tag", "limit", "offset").
	WithSummary("Enumerate published documents").
	WithTags("Published").
	WithErrors(rocco.ErrUnauthorized).
	WithAuthentication()

// ListPublishedFolder lists the published documents directly inside a folder —
// the render path for a section page or nav level. The path query parameter is
// the folder ("guides" or "guides/setup"); absent means the app root. One term
// query on the materialized parent_path, never a tree walk.
var ListPublishedFolder = rocco.GET("/published/apps/{app_id}/folder",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		docs, total, err := reads.ListFolder(ctx, req.Params.Path["app_id"], req.Params.Query["path"], limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithPathParams("app_id").
	WithQueryParams("path", "limit", "offset").
	WithSummary("List a folder's published documents").
	WithTags("Published").
	WithErrors(rocco.ErrUnauthorized).
	WithAuthentication()

// SearchDocuments runs a full-text search over an app's published content.
var SearchDocuments = rocco.GET("/published/apps/{app_id}/search",
	func(req *rocco.Request[rocco.NoBody]) (wire.PublishedDocumentListResponse, error) {
		query := req.Params.Query["q"]
		if query == "" {
			return wire.PublishedDocumentListResponse{}, rocco.ErrBadRequest.WithMessage("q query parameter required")
		}
		reads := sum.MustUse[contracts.Reads](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		docs, total, err := reads.Search(ctx, req.Params.Path["app_id"], query, limit, offset)
		if err != nil {
			return wire.PublishedDocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.IndexesToListResponse(docs, total, limit, offset), nil
	}).WithPathParams("app_id").
	WithQueryParams("q", "limit", "offset").
	WithSummary("Full-text search over published documents").
	WithTags("Published").
	WithErrors(rocco.ErrBadRequest, rocco.ErrUnauthorized).
	WithAuthentication()
