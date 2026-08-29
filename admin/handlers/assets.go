package handlers

import (
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/admin/contracts"
	"github.com/zoobz-io/barbara/admin/transformers"
	"github.com/zoobz-io/barbara/admin/wire"
	"github.com/zoobz-io/barbara/internal/auth"
)

// maxAssetBytes caps a single asset upload. Assets are images and other blobs
// markdown references, not archives — 50 MiB is generous headroom.
const maxAssetBytes = 50 << 20

// UploadAsset stores (or overwrites) an asset by key for the request's tenant.
// The body is the raw bytes; the key is a query parameter, since asset keys are
// opaque and may contain slashes. Putting an existing key overwrites it.
var UploadAsset = rocco.PUT("/assets/object",
	func(req *rocco.Request[rocco.RawBody]) (wire.AssetResponse, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		asset, err := assets.Put(ctx, req.Params.Query["key"], req.Body.ContentType, req.Body.Data)
		if err != nil {
			return wire.AssetResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AssetToResponse(asset), nil
	}).WithQueryParams("key").
	WithSummary("Upload an asset (overwrites an existing key)").
	WithTags("Assets").
	WithMaxBodySize(maxAssetBytes).
	WithErrors(rocco.ErrBadRequest, rocco.ErrUnauthorized).
	WithAuthentication()

// GetAsset downloads an asset's bytes by key, served with its stored content
// type. The response bypasses the JSON codec — the blob is written as-is.
var GetAsset = rocco.GET("/assets/object",
	func(req *rocco.Request[rocco.NoBody]) (rocco.Blob, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		asset, err := assets.Get(ctx, req.Params.Query["key"])
		if err != nil {
			return rocco.Blob{}, transformers.ErrorToResponse(err)
		}
		return rocco.Blob{ContentType: asset.ContentType, Data: asset.Data}, nil
	}).WithQueryParams("key").
	WithSummary("Download an asset").
	WithTags("Assets").
	WithMediaTypes("application/octet-stream", "image/png", "image/jpeg", "application/pdf").
	WithErrors(rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()

// ListAssets returns metadata for the request tenant's assets (no bytes).
var ListAssets = rocco.GET("/assets",
	func(req *rocco.Request[rocco.NoBody]) (wire.AssetListResponse, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		list, err := assets.List(ctx)
		if err != nil {
			return wire.AssetListResponse{}, transformers.ErrorToResponse(err)
		}
		return transformers.AssetsToListResponse(list), nil
	}).WithSummary("List assets").
	WithTags("Assets").
	WithErrors(rocco.ErrUnauthorized).
	WithAuthentication()

// DeleteAsset removes an asset by key for the request's tenant.
var DeleteAsset = rocco.DELETE("/assets/object",
	func(req *rocco.Request[rocco.NoBody]) (wire.AssetResponse, error) {
		assets := sum.MustUse[contracts.Assets](req.Context)
		ctx := auth.WithPrincipal(req.Context, req.Identity)
		if err := assets.Delete(ctx, req.Params.Query["key"]); err != nil {
			return wire.AssetResponse{}, transformers.ErrorToResponse(err)
		}
		return wire.AssetResponse{}, nil
	}).WithQueryParams("key").
	WithSummary("Delete an asset").
	WithTags("Assets").
	WithSuccessStatus(204).
	WithErrors(rocco.ErrNotFound, rocco.ErrUnauthorized).
	WithAuthentication()
