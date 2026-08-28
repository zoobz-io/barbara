// Package handlers defines the admin (authoring) HTTP endpoints. Handlers are
// thin: resolve the contract, bridge the request identity into the context so
// the store is tenant-scoped, call the contract, and transform the result.
package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/admin/contracts"
	"github.com/zoobz-io/barbara/admin/transformers"
	"github.com/zoobz-io/barbara/admin/wire"
	"github.com/zoobz-io/barbara/internal/auth"
)

// CreateDocument creates a document for the request's tenant.
var CreateDocument = rocco.POST("/documents",
	func(req *rocco.Request[wire.CreateDocumentRequest]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := docs.Create(ctx, req.Body.Key)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithSummary("Create a document").
	WithTags("Documents").
	WithSuccessStatus(201).
	WithErrors(rocco.ErrConflict, rocco.ErrUnauthorized).
	WithAuthentication()

// GetDocument returns a document by ID.
var GetDocument = rocco.GET("/documents/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := docs.Get(ctx, req.Params.Path["id"])
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithSummary("Get a document by ID").
	WithTags("Documents").
	WithErrors(rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// ListDocuments returns the tenant's documents, paginated.
var ListDocuments = rocco.GET("/documents",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentListResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := transformers.Pagination(req.Params.Query)
		list, err := docs.List(ctx, limit, offset)
		if err != nil {
			return wire.DocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentsToListResponse(list, limit, offset), nil
	}).WithQueryParams("limit", "offset").
	WithSummary("List documents").
	WithTags("Documents").
	WithErrors(rocco.ErrUnauthorized).
	WithAuthentication()

// RenameDocument changes a document's key.
var RenameDocument = rocco.PATCH("/documents/{id}",
	func(req *rocco.Request[wire.RenameDocumentRequest]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := docs.Rename(ctx, req.Params.Path["id"], req.Body.Key)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithSummary("Rename a document").
	WithTags("Documents").
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrUnauthorized).
	WithAuthentication()

// DeleteDocument removes an unpublished document.
var DeleteDocument = rocco.DELETE("/documents/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		if err := docs.Delete(ctx, req.Params.Path["id"]); err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return wire.DocumentResponse{}, nil
	}).WithPathParams("id").
	WithSummary("Delete an unpublished document").
	WithTags("Documents").
	WithSuccessStatus(204).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrUnauthorized).
	WithAuthentication()
