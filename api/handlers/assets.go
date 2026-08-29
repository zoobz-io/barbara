package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/api/transformers"
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/internal/auth"
)

// maxAssetBytes caps a single asset upload. Assets are images and other blobs
// markdown references, not archives — 50 MiB is generous headroom.
const maxAssetBytes = 50 << 20

// UploadAsset stores (or overwrites) an asset by key for the app. The body is
// the raw bytes; the key is a query parameter, since asset keys are opaque and
// may contain slashes. Putting an existing key overwrites it.
var UploadAsset = rocco.PUT("/apps/{app_id}/assets/object",
	func(req *rocco.Request[rocco.RawBody]) (wire.AssetResponse, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		asset, err := assets.Put(ctx, req.Params.Path["app_id"], req.Params.Query["key"], req.Body.ContentType, req.Body.Data)
		if err != nil {
			return wire.AssetResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AssetToResponse(asset), nil
	}).WithPathParams("app_id").
	WithQueryParams("key").
	WithSummary("Upload an asset (overwrites an existing key)").
	WithTags("Assets").
	WithMaxBodySize(maxAssetBytes).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrBadRequest, rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetAsset downloads an asset's bytes by key, served with its stored content
// type. The response bypasses the JSON codec — the blob is written as-is.
var GetAsset = rocco.GET("/apps/{app_id}/assets/object",
	func(req *rocco.Request[rocco.NoBody]) (rocco.Blob, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		asset, err := assets.Get(ctx, req.Params.Path["app_id"], req.Params.Query["key"])
		if err != nil {
			return rocco.Blob{}, transformers.ErrorToResponse(err)
		}
		return rocco.Blob{ContentType: asset.ContentType, Data: asset.Data}, nil
	}).WithPathParams("app_id").
	WithQueryParams("key").
	WithSummary("Download an asset").
	WithTags("Assets").
	WithMediaTypes("application/octet-stream", "image/png", "image/jpeg", "application/pdf").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// ListAssets returns metadata for the app's assets (no bytes). An optional
// prefix query parameter narrows the listing to keys under it — the folder
// view, since an asset folder is a key prefix by convention.
var ListAssets = rocco.GET("/apps/{app_id}/assets",
	func(req *rocco.Request[rocco.NoBody]) (wire.AssetListResponse, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		list, err := assets.List(ctx, req.Params.Path["app_id"], req.Params.Query["prefix"])
		if err != nil {
			return wire.AssetListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AssetsToListResponse(list), nil
	}).WithPathParams("app_id").
	WithQueryParams("prefix").
	WithSummary("List assets").
	WithTags("Assets").
	WithScopes(auth.ScopeDocumentsRead).
	WithErrors(rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// DeleteAsset removes an asset by key for the app.
var DeleteAsset = rocco.DELETE("/apps/{app_id}/assets/object",
	func(req *rocco.Request[rocco.NoBody]) (wire.AssetResponse, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		if err := assets.Delete(ctx, req.Params.Path["app_id"], req.Params.Query["key"]); err != nil {
			return wire.AssetResponse{}, transformers.ErrorToResponse(err)
		}
		return wire.AssetResponse{}, nil
	}).WithPathParams("app_id").
	WithQueryParams("key").
	WithSummary("Delete an asset").
	WithTags("Assets").
	WithSuccessStatus(204).
	WithScopes(auth.ScopeDocumentsWrite).
	WithErrors(rocco.ErrNotFound, rocco.ErrForbidden, rocco.ErrUnauthorized).
	WithAuthentication()

// GetPublishedAsset serves an asset's bytes on the published surface — the
// read a live site uses for the images its markdown references. Assets are not
// in OpenSearch and not in releases, so this is a bucket read of the live
// object: the only version there is. Plain tenant authentication, no authoring
// scope — it is the site's render path, like the published document reads.
var GetPublishedAsset = rocco.GET("/published/apps/{app_id}/assets/object",
	func(req *rocco.Request[rocco.NoBody]) (rocco.Blob, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		asset, err := assets.Get(ctx, req.Params.Path["app_id"], req.Params.Query["key"])
		if err != nil {
			return rocco.Blob{}, transformers.ErrorToResponse(err)
		}
		return rocco.Blob{ContentType: asset.ContentType, Data: asset.Data}, nil
	}).WithPathParams("app_id").
	WithQueryParams("key").
	WithSummary("Download a published app's asset").
	WithTags("Assets").
	WithMediaTypes("application/octet-stream", "image/png", "image/jpeg", "application/pdf").
	WithErrors(rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()
