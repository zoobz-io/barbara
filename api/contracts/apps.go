package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Apps is the authoring view of the app — the release unit. Every method is
// tenant-scoped via the request context.
type Apps interface {
	// Create makes a new app with the given name.
	Create(ctx context.Context, name string) (*models.App, error)
	// Get retrieves an app by ID.
	Get(ctx context.Context, id string) (*models.App, error)
	// List returns the tenant's apps, paginated.
	List(ctx context.Context, limit, offset int) ([]*models.App, error)
	// Rename changes an app's name.
	Rename(ctx context.Context, id, newName string) (*models.App, error)
	// Delete removes an app that has no release.
	Delete(ctx context.Context, id string) error
}
