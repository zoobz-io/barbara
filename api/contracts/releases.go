package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Releases is the authoring view of releases — the immutable publish snapshots.
// Every method is scoped to the tenant (context) and the app (argument).
type Releases interface {
	// Cut snapshots the whole live tree into a new release and moves the app's
	// pointer to it.
	Cut(ctx context.Context, appID string) (*models.Release, error)
	// List returns the app's releases, newest first, paginated.
	List(ctx context.Context, appID string, limit, offset int) ([]*models.Release, error)
	// Get returns a release with its entries.
	Get(ctx context.Context, appID, releaseID string) (*models.Release, []*models.ReleaseEntry, error)
	// Rollback cuts a new release copying an old release's entries forward.
	Rollback(ctx context.Context, appID, releaseID string) (*models.Release, error)
}
