package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/internal/auth"
)

// AddDocumentTag adds a tag to a document. On a published document this
// re-projects into OpenSearch without moving the published pointer.
var AddDocumentTag = rocco.POST("/documents/{id}/tags",
	func(req *rocco.Request[wire.AddTagRequest]) (wire.DocumentResponse, error) {
		if req.Body.Tag == "" {
			return wire.DocumentResponse{}, rocco.ErrBadRequest.WithMessage("tag is required")
		}
		tagging := sum.MustUse[contracts.Tagging](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := tagging.AddTag(ctx, req.Params.Path["id"], req.Body.Tag)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithSummary("Add a tag to a document").
	WithTags("Documents").
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// RemoveDocumentTag removes a tag from a document. The tag is a query parameter
// rather than a path segment, since a tag is an opaque label that may carry
// characters awkward in a path. On a published document this re-projects into
// OpenSearch without moving the published pointer.
var RemoveDocumentTag = rocco.DELETE("/documents/{id}/tags",
	func(req *rocco.Request[rocco.NoBody]) (wire.DocumentResponse, error) {
		tag := req.Params.Query["tag"]
		if tag == "" {
			return wire.DocumentResponse{}, rocco.ErrBadRequest.WithMessage("tag query parameter required")
		}
		tagging := sum.MustUse[contracts.Tagging](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		doc, err := tagging.RemoveTag(ctx, req.Params.Path["id"], tag)
		if err != nil {
			return wire.DocumentResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.DocumentToResponse(doc), nil
	}).WithPathParams("id").
	WithQueryParams("tag").
	WithSummary("Remove a tag from a document").
	WithTags("Documents").
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()
