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

// CreateApp creates an app for the request's tenant.
var CreateApp = rocco.POST("/apps",
	func(req *rocco.Request[wire.CreateAppRequest]) (wire.AppResponse, error) {
		apps := sum.MustUse[contracts.Apps](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		app, err := apps.Create(ctx, req.Body.Name)
		if err != nil {
			return wire.AppResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AppToResponse(app), nil
	}).WithSummary("Create an app").
	WithTags("Apps").
	WithSuccessStatus(201).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetApp returns an app by ID.
var GetApp = rocco.GET("/apps/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.AppResponse, error) {
		apps := sum.MustUse[contracts.Apps](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		app, err := apps.Get(ctx, req.Params.Path["id"])
		if err != nil {
			return wire.AppResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AppToResponse(app), nil
	}).WithPathParams("id").
	WithSummary("Get an app by ID").
	WithTags("Apps").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListApps returns the tenant's apps, paginated.
var ListApps = rocco.GET("/apps",
	func(req *rocco.Request[rocco.NoBody]) (wire.AppListResponse, error) {
		apps := sum.MustUse[contracts.Apps](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		limit, offset := dbtransformers.Pagination(req.Params.Query)
		list, err := apps.List(ctx, limit, offset)
		if err != nil {
			return wire.AppListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AppsToListResponse(list, limit, offset), nil
	}).WithQueryParams("limit", "offset").
	WithSummary("List apps").
	WithTags("Apps").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// RenameApp changes an app's name.
var RenameApp = rocco.PATCH("/apps/{id}",
	func(req *rocco.Request[wire.RenameAppRequest]) (wire.AppResponse, error) {
		apps := sum.MustUse[contracts.Apps](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		app, err := apps.Rename(ctx, req.Params.Path["id"], req.Body.Name)
		if err != nil {
			return wire.AppResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AppToResponse(app), nil
	}).WithPathParams("id").
	WithSummary("Rename an app").
	WithTags("Apps").
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// DeleteApp removes an app that has no release.
var DeleteApp = rocco.DELETE("/apps/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.AppResponse, error) {
		apps := sum.MustUse[contracts.Apps](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		if err := apps.Delete(ctx, req.Params.Path["id"]); err != nil {
			return wire.AppResponse{}, transformers.ErrorToResponse(err)
		}
		return wire.AppResponse{}, nil
	}).WithPathParams("id").
	WithSummary("Delete an app that has no release").
	WithTags("Apps").
	WithSuccessStatus(204).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()
