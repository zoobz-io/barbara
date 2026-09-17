package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Assets is the authoring view of binary assets in object storage. Every
// method is tenant-scoped via the request context and app-scoped by parameter;
// the key is unique per app, and putting the same key overwrites (assets are
// not versioned). Assets live outside the collection tree and outside
// releases — a folder is a key prefix by convention.
type Assets interface {
	// Put stores data at key for the app, overwriting any existing asset, and
	// returns the stored metadata. The app must exist for the tenant.
	Put(ctx context.Context, appID, key, contentType string, data []byte) (*models.Asset, error)
	// Get returns the app's asset at key, bytes included.
	Get(ctx context.Context, appID, key string) (*models.Asset, error)
	// List returns metadata for the app's assets, without the bytes. A
	// non-empty keyPrefix narrows to keys under it.
	List(ctx context.Context, appID, keyPrefix string) ([]*models.Asset, error)
	// ListFolder returns one level of the app's asset tree — the direct
	// subfolders (with counts) and the assets directly at folderPath. An
	// empty folderPath is the root.
	ListFolder(ctx context.Context, appID, folderPath string) (*models.AssetLevel, error)
	// CreateFolder makes folderPath an explicit folder of the app — a level
	// that lists before any object lands in it — and returns its level.
	CreateFolder(ctx context.Context, appID, folderPath string) (*models.AssetLevel, error)
	// Stats returns the app-level bookkeeping view: the root rollup, the
	// per-kind breakdown, and the daily write series.
	Stats(ctx context.Context, appID string) (*models.AssetStats, error)
	// Move changes the app's asset at key to newKey — another folder,
	// another name, or both. The destination must be free.
	Move(ctx context.Context, appID, key, newKey string) (*models.Asset, error)
	// Delete removes the app's asset at key.
	Delete(ctx context.Context, appID, key string) error
}
