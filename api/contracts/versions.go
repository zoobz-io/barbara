package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Versions is the authoring view of immutable document versions. Every method
// is tenant-scoped via the request context.
type Versions interface {
	// Save appends a new version of the document's content, allocating the next
	// version number.
	Save(ctx context.Context, documentID, content string) (*models.Version, error)
	// List returns a document's versions, newest first, paginated.
	List(ctx context.Context, documentID string, limit, offset int) ([]*models.Version, error)
	// Get retrieves a version by ID.
	Get(ctx context.Context, id string) (*models.Version, error)
}
