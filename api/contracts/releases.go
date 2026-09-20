package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Releases is the authoring view of releases — the immutable publish snapshots.
// Every method is scoped to the tenant (context) and the app (argument).
type Releases interface {
	// Cut snapshots the whole live tree into a new release and moves the app's
	// pointer to it. label is optional ("" for none) and is fixed at cut time.
	Cut(ctx context.Context, appID, label string) (*models.Release, error)
	// List returns the app's releases, newest first, paginated.
	List(ctx context.Context, appID string, limit, offset int) ([]*models.Release, error)
	// Total returns how many releases the app has.
	Total(ctx context.Context, appID string) (int64, error)
	// Get returns a release with its entries.
	Get(ctx context.Context, appID, releaseID string) (*models.Release, []*models.ReleaseEntry, error)
	// Changes returns a release with its changes against the previous release.
	Changes(ctx context.Context, appID, releaseID string) (*models.Release, []*models.ReleaseChange, error)
	// Rollback cuts a new release copying an old release's entries forward.
	// label is optional.
	Rollback(ctx context.Context, appID, releaseID, label string) (*models.Release, error)
}
