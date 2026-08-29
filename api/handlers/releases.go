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

// CutRelease snapshots the app's live tree into a new release and moves the
// pointer. Sits behind the publish scope.
var CutRelease = rocco.POST("/apps/{app_id}/releases",
	func(req *rocco.Request[rocco.NoBody]) (wire.ReleaseResponse, error) {
		releases := sum.MustUse[contracts.Releases](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		release, err := releases.Cut(ctx, req.Params.Path["app_id"])
		if err != nil {
			return wire.ReleaseResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.ReleaseToResponse(release), nil
	}).WithPathParams("app_id").
	WithSummary("Cut a release").
	WithTags("Releases").
	WithSuccessStatus(201).
	WithScopes(auth.ScopeDocumentsPublish).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListReleases returns the app's releases, newest first.
var ListReleases = rocco.GET("/apps/{app_id}/releases",
	func(req *rocco.Request[rocco.NoBody]) (wire.ReleaseListResponse, error) {
		releases := sum.MustUse[contracts.Releases](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		list, err := releases.List(ctx, req.Params.Path["app_id"], limit, offset)
		if err != nil {
			return wire.ReleaseListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.ReleasesToListResponse(list, limit, offset), nil
	}).WithPathParams("app_id").
	WithQueryParams("limit", "offset").
	WithSummary("List releases").
	WithTags("Releases").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetRelease returns a release with its entries.
var GetRelease = rocco.GET("/apps/{app_id}/releases/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.ReleaseWithEntriesResponse, error) {
		releases := sum.MustUse[contracts.Releases](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		release, entries, err := releases.Get(ctx, req.Params.Path["app_id"], req.Params.Path["id"])
		if err != nil {
			return wire.ReleaseWithEntriesResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.ReleaseWithEntriesToResponse(release, entries), nil
	}).WithPathParams("app_id", "id").
	WithSummary("Get a release with its entries").
	WithTags("Releases").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// RollbackRelease cuts a new release copying an old release's entries forward.
// Sits behind the publish scope.
var RollbackRelease = rocco.POST("/apps/{app_id}/releases/{id}/rollback",
	func(req *rocco.Request[rocco.NoBody]) (wire.ReleaseResponse, error) {
		releases := sum.MustUse[contracts.Releases](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		release, err := releases.Rollback(ctx, req.Params.Path["app_id"], req.Params.Path["id"])
		if err != nil {
			return wire.ReleaseResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.ReleaseToResponse(release), nil
	}).WithPathParams("app_id", "id").
	WithSummary("Roll back to a release").
	WithTags("Releases").
	WithSuccessStatus(201).
	WithScopes(auth.ScopeDocumentsPublish).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()
