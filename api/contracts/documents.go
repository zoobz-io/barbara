package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Documents is the authoring view of the logical document. Every method is
// tenant-scoped via the request context; create and move are additionally
// app-scoped.
type Documents interface {
	// Create places a new document under a collection (nil = app root) in the
	// app, with the given name; the key is materialized from the tree.
	Create(ctx context.Context, appID string, collectionID *string, name string) (*models.Document, error)
	// Get retrieves a document by ID.
	Get(ctx context.Context, id string) (*models.Document, error)
	// GetWithHead retrieves a document with its head (latest) version — the
	// single-call read behind opening a document for editing.
	GetWithHead(ctx context.Context, id string) (*models.DocumentHead, error)
	// List returns the tenant's documents, paginated.
	List(ctx context.Context, limit, offset int) ([]*models.Document, error)
	// ListByTag returns the tenant's documents carrying the given tag, paginated.
	ListByTag(ctx context.Context, tag string, limit, offset int) ([]*models.Document, error)
	// Move reparents and/or renames a document, rewriting its key.
	Move(ctx context.Context, appID, id string, newCollectionID *string, newName string) (*models.Document, error)
	// Delete removes a document absent from the current release (hard-deleting an
	// unreferenced one, soft-deleting one a historical release references).
	Delete(ctx context.Context, id string) error
}
