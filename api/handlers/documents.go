package handlers

import (
	"context"

	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
	dbtransformers "github.com/zoobz-io/barbara/database/transformers"
	"github.com/zoobz-io/barbara/internal/auth"
)

// documentResponse builds a document response, deriving its release-based status
// through the store. Shared by every handler that returns a single document.
func documentResponse(ctx context.Context, docs contracts.Documents, doc *models.Document) (wire.DocumentResponse, error) {
	status, err := docs.Status(ctx, doc)
	if err != nil {
		return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
	}
	return transformers.DocumentToResponse(doc, status), nil
}

// CreateDocument places a document in the tree — under a collection (or the app
// root) in the app.
var CreateDocument = rocco.POST("/apps/{app_id}/documents",
	func(req *rocco.Request[wire.CreateDocumentRequest]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := docs.Create(ctx, req.Params.Path["app_id"], req.Body.CollectionID, req.Body.Name)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return documentResponse(ctx, docs, doc)
	}).WithPathParams("app_id").
	WithSummary("Create a document").
	WithTags("Documents").
	WithSuccessStatus(201).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetDocument returns a document by ID, carrying its draft/published status.
var GetDocument = rocco.GET("/documents/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := docs.Get(ctx, req.Params.Path["id"])
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return documentResponse(ctx, docs, doc)
	}).WithPathParams("id").
	WithSummary("Get a document by ID").
	WithTags("Documents").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetDocumentContent returns a document together with its head version's content
// — the single request an editing client makes to open a document. The content
// block is null when the document has no versions yet (an empty document, not a
// 404).
var GetDocumentContent = rocco.GET("/documents/{id}/content",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentContentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		dh, err := docs.GetWithHead(ctx, req.Params.Path["id"])
		if err != nil {
			return wire.DocumentContentResponse{}, transformers.ErrorToResponse(err)
		}
		status, serr := docs.Status(ctx, dh.Document)
		if serr != nil {
			return wire.DocumentContentResponse{}, transformers.ErrorToResponse(serr)
		}
		return transformers.DocumentContentToResponse(dh, status), nil
	}).WithPathParams("id").
	WithSummary("Get a document with its head version content").
	WithTags("Documents").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListDocuments returns the tenant's documents, paginated. An optional tag
// query parameter filters to documents carrying that tag.
var ListDocuments = rocco.GET("/documents",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentListResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		var (
			list []*models.Document
			err  error
		)
		if tag := req.Params.Query["tag"]; tag != "" {
			list, err = docs.ListByTag(ctx, tag, limit, offset)
		} else {
			list, err = docs.List(ctx, limit, offset)
		}
		if err != nil {
			return wire.DocumentListResponse{}, transformers.ErrorToResponse(err)
		}
		statuses, serr := docs.Statuses(ctx, list)
		if serr != nil {
			return wire.DocumentListResponse{}, transformers.ErrorToResponse(serr)
		}
		return transformers.DocumentsToListResponse(list, statuses, limit, offset), nil
	}).WithQueryParams("tag", "limit", "offset").
	WithSummary("List documents").
	WithTags("Documents").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// MoveDocument reparents and/or renames a document, rewriting its key.
var MoveDocument = rocco.POST("/apps/{app_id}/documents/{id}/move",
	func(req *rocco.Request[wire.MoveDocumentRequest]) (wire.DocumentResponse, error) {
		docs := sum.MustUse[contracts.Documents](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := docs.Move(ctx, req.Params.Path["app_id"], req.Params.Path["id"], req.Body.CollectionID, req.Body.Name)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return documentResponse(ctx, docs, doc)
	}).WithPathParams("app_id", "id").
	WithSummary("Move a document to a new collection or name").
	WithTags("Documents").
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
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
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()
