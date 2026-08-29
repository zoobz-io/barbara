package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// AssetToResponse maps an asset model to its authoring metadata response (no bytes).
func AssetToResponse(a *models.Asset) wire.AssetResponse {
	return wire.AssetResponse{
		Key:         a.Key,
		ContentType: a.ContentType,
		Size:        a.Size,
	}
}

// AssetsToListResponse maps a slice of assets to the authoring list response.
func AssetsToListResponse(assets []*models.Asset) wire.AssetListResponse {
	out := wire.AssetListResponse{
		Assets: make([]wire.AssetResponse, len(assets)),
		Total:  len(assets),
	}
	for i, a := range assets {
		out.Assets[i] = AssetToResponse(a)
	}
	return out
}
