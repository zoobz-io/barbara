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
	// non-empty keyPrefix narrows to keys under it (the folder view).
	List(ctx context.Context, appID, keyPrefix string) ([]*models.Asset, error)
	// Delete removes the app's asset at key.
	Delete(ctx context.Context, appID, key string) error
}
