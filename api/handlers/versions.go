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

// SaveVersion appends a new version of a document's content.
var SaveVersion = rocco.POST("/documents/{document_id}/versions",
	func(req *rocco.Request[wire.SaveVersionRequest]) (wire.VersionResponse, error) {
		versions := sum.MustUse[contracts.Versions](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		v, err := versions.Save(ctx, req.Params.Path["document_id"], req.Body.Content, *req.Body.BaseVersion)
		if err != nil {
			return wire.VersionResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.VersionToResponse(v), nil
	}).WithPathParams("document_id").
	WithSummary("Save a new version of a document").
	WithTags("Versions").
	WithSuccessStatus(201).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListVersions returns a document's versions, newest first.
var ListVersions = rocco.GET("/documents/{document_id}/versions",
	func(req *rocco.Request[rocco.NoBody]) (wire.VersionListResponse, error) {
		versions := sum.MustUse[contracts.Versions](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		list, err := versions.List(ctx, req.Params.Path["document_id"], limit, offset)
		if err != nil {
			return wire.VersionListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.VersionsToListResponse(list, limit, offset), nil
	}).WithPathParams("document_id").
	WithQueryParams("limit", "offset").
	WithSummary("List a document's versions").
	WithTags("Versions").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetVersion returns a single version by ID.
var GetVersion = rocco.GET("/versions/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.VersionResponse, error) {
		versions := sum.MustUse[contracts.Versions](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		v, err := versions.Get(ctx, req.Params.Path["id"])
		if err != nil {
			return wire.VersionResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.VersionToResponse(v), nil
	}).WithPathParams("id").
	WithSummary("Get a version by ID").
	WithTags("Versions").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()
