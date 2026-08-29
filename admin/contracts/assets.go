package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Assets is the authoring view of binary assets in object storage. Every method
// is tenant-scoped via the request context; the key is unique per tenant, and
// putting the same key overwrites (assets are not versioned).
type Assets interface {
	// Put stores data at key, overwriting any existing asset, and returns the
	// stored metadata.
	Put(ctx context.Context, key, contentType string, data []byte) (*models.Asset, error)
	// Get returns the asset at key, bytes included.
	Get(ctx context.Context, key string) (*models.Asset, error)
	// List returns metadata for the tenant's assets, without the bytes.
	List(ctx context.Context) ([]*models.Asset, error)
	// Delete removes the asset at key.
	Delete(ctx context.Context, key string) error
}
