package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/internal/auth"
)

// CreateCollection creates a collection under a parent (or the app root).
var CreateCollection = rocco.POST("/apps/{app_id}/collections",
	func(req *rocco.Request[wire.CreateCollectionRequest]) (wire.CollectionResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		col, err := cols.Create(ctx, req.Params.Path["app_id"], req.Body.ParentID, req.Body.Name)
		if err != nil {
			return wire.CollectionResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.CollectionToResponse(col), nil
	}).WithPathParams("app_id").
	WithSummary("Create a collection").
	WithTags("Collections").
	WithSuccessStatus(201).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetCollection returns a collection by ID within the app.
var GetCollection = rocco.GET("/apps/{app_id}/collections/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.CollectionResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		col, err := cols.Get(ctx, req.Params.Path["app_id"], req.Params.Path["id"])
		if err != nil {
			return wire.CollectionResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.CollectionToResponse(col), nil
	}).WithPathParams("app_id", "id").
	WithSummary("Get a collection by ID").
	WithTags("Collections").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListAppRootContents lists the app root's subcollections and documents.
var ListAppRootContents = rocco.GET("/apps/{app_id}/contents",
	func(req *rocco.Request[rocco.NoBody]) (wire.CollectionContentsResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		contents, err := cols.ListContents(ctx, req.Params.Path["app_id"], nil)
		if err != nil {
			return wire.CollectionContentsResponse{}, transformers.ErrorToResponse(err)
		}
		statuses, serr := sum.MustUse[contracts.Documents](req.Context).Statuses(ctx, contents.Documents)
		if serr != nil {
			return wire.CollectionContentsResponse{}, transformers.ErrorToResponse(serr)
		}
		return transformers.CollectionContentsToResponse(contents, statuses), nil
	}).WithPathParams("app_id").
	WithSummary("List the app root contents").
	WithTags("Collections").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListCollectionContents lists a collection's subcollections and documents.
var ListCollectionContents = rocco.GET("/apps/{app_id}/collections/{id}/contents",
	func(req *rocco.Request[rocco.NoBody]) (wire.CollectionContentsResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		id := req.Params.Path["id"]
		contents, err := cols.ListContents(ctx, req.Params.Path["app_id"], &id)
		if err != nil {
			return wire.CollectionContentsResponse{}, transformers.ErrorToResponse(err)
		}
		statuses, serr := sum.MustUse[contracts.Documents](req.Context).Statuses(ctx, contents.Documents)
		if serr != nil {
			return wire.CollectionContentsResponse{}, transformers.ErrorToResponse(serr)
		}
		return transformers.CollectionContentsToResponse(contents, statuses), nil
	}).WithPathParams("app_id", "id").
	WithSummary("List a collection's contents").
	WithTags("Collections").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// RenameCollection changes a collection's name.
var RenameCollection = rocco.PATCH("/apps/{app_id}/collections/{id}",
	func(req *rocco.Request[wire.RenameCollectionRequest]) (wire.CollectionResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		col, err := cols.Rename(ctx, req.Params.Path["app_id"], req.Params.Path["id"], req.Body.Name)
		if err != nil {
			return wire.CollectionResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.CollectionToResponse(col), nil
	}).WithPathParams("app_id", "id").
	WithSummary("Rename a collection").
	WithTags("Collections").
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// MoveCollection reparents a collection (or moves it to the app root).
var MoveCollection = rocco.POST("/apps/{app_id}/collections/{id}/move",
	func(req *rocco.Request[wire.MoveCollectionRequest]) (wire.CollectionResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		col, err := cols.Move(ctx, req.Params.Path["app_id"], req.Params.Path["id"], req.Body.ParentID)
		if err != nil {
			return wire.CollectionResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.CollectionToResponse(col), nil
	}).WithPathParams("app_id", "id").
	WithSummary("Move a collection to a new parent").
	WithTags("Collections").
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// DeleteCollection removes an empty collection.
var DeleteCollection = rocco.DELETE("/apps/{app_id}/collections/{id}",
	func(req *rocco.Request[rocco.NoBody]) (wire.CollectionResponse, error) {
		cols := sum.MustUse[contracts.Collections](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		if err := cols.Delete(ctx, req.Params.Path["app_id"], req.Params.Path["id"]); err != nil {
			return wire.CollectionResponse{}, transformers.ErrorToResponse(err)
		}
		return wire.CollectionResponse{}, nil
	}).WithPathParams("app_id", "id").
	WithSummary("Delete an empty collection").
	WithTags("Collections").
	WithSuccessStatus(204).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrConflict, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()
