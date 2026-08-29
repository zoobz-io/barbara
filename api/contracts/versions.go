package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Versions is the authoring view of immutable document versions. Every method
// is tenant-scoped via the request context.
type Versions interface {
	// Save appends a new version of the document's content, allocating the next
	// version number — only if baseVersion is still the head (optimistic
	// concurrency); otherwise it returns a *stores.VersionConflictError.
	Save(ctx context.Context, documentID, content string, baseVersion int) (*models.Version, error)
	// List returns a document's versions, newest first, paginated.
	List(ctx context.Context, documentID string, limit, offset int) ([]*models.Version, error)
	// Get retrieves a version by ID.
	Get(ctx context.Context, id string) (*models.Version, error)
}
