package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Documents is the authoring view of the logical document. Every method is
// tenant-scoped via the request context.
type Documents interface {
	// Create makes a new document with the given key.
	Create(ctx context.Context, key string) (*models.Document, error)
	// Get retrieves a document by ID.
	Get(ctx context.Context, id string) (*models.Document, error)
	// List returns the tenant's documents, paginated.
	List(ctx context.Context, limit, offset int) ([]*models.Document, error)
	// ListByTag returns the tenant's documents carrying the given tag, paginated.
	ListByTag(ctx context.Context, tag string, limit, offset int) ([]*models.Document, error)
	// Rename changes a document's key, freeing the old one.
	Rename(ctx context.Context, id, newKey string) (*models.Document, error)
	// Delete removes an unpublished document (cascading to its versions).
	Delete(ctx context.Context, id string) error
}
