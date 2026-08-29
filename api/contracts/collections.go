package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Collections is the authoring view of the folder tree. Every method is scoped
// to the tenant (context) and the app (argument).
type Collections interface {
	// Create makes a collection under parentID (nil = app root) in the app.
	Create(ctx context.Context, appID string, parentID *string, name string) (*models.Collection, error)
	// Get retrieves a collection by ID within the app.
	Get(ctx context.Context, appID, id string) (*models.Collection, error)
	// ListContents returns a collection's subcollections and documents in one
	// call (collectionID nil = the app root).
	ListContents(ctx context.Context, appID string, collectionID *string) (*models.CollectionContents, error)
	// Rename changes a collection's name, rewriting descendant document keys.
	Rename(ctx context.Context, appID, id, newName string) (*models.Collection, error)
	// Move reparents a collection under newParentID (nil = app root), rewriting
	// descendant document keys.
	Move(ctx context.Context, appID, id string, newParentID *string) (*models.Collection, error)
	// Delete removes an empty collection.
	Delete(ctx context.Context, appID, id string) error
}
