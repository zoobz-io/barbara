package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/internal/auth"
)

// PublishDocument points a document at a version and projects it into search.
var PublishDocument = rocco.POST("/documents/{id}/publish",
	func(req *rocco.Request[wire.PublishRequest]) (wire.DocumentResponse, error) {
		publishing := sum.MustUse[contracts.Publishing](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := publishing.Publish(ctx, req.Params.Path["id"], req.Body.VersionID)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithSummary("Publish a document version").
	WithTags("Publishing").
	WithErrors(rocco.ErrNotFound, rocco.ErrBadRequest, rocco.ErrUnauthorized).
	WithAuthentication()

// UnpublishDocument clears a document's published pointer and removes its entry.
var UnpublishDocument = rocco.POST("/documents/{id}/unpublish",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentResponse, error) {
		publishing := sum.MustUse[contracts.Publishing](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := publishing.Unpublish(ctx, req.Params.Path["id"])
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithSummary("Unpublish a document").
	WithTags("Publishing").
	WithErrors(rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// RollbackDocument republishes an older version of a document.
var RollbackDocument = rocco.POST("/documents/{id}/rollback",
	func(req *rocco.Request[wire.PublishRequest]) (wire.DocumentResponse, error) {
		publishing := sum.MustUse[contracts.Publishing](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := publishing.Rollback(ctx, req.Params.Path["id"], req.Body.VersionID)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithSummary("Roll back to an older document version").
	WithTags("Publishing").
	WithErrors(rocco.ErrNotFound, rocco.ErrBadRequest, rocco.ErrUnauthorized).
	WithAuthentication()
