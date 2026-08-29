package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Publishing is the authoring view of the publish lifecycle. Each method is an
// atomic pointer-move plus an enqueued OpenSearch write, tenant-scoped via the
// request context, and returns the updated document.
type Publishing interface {
	// Publish points the document at the given version and projects it.
	Publish(ctx context.Context, documentID, versionID string) (*models.Document, error)
	// Unpublish clears the document's published pointer and removes its entry.
	Unpublish(ctx context.Context, documentID string) (*models.Document, error)
	// Rollback republishes an older version.
	Rollback(ctx context.Context, documentID, versionID string) (*models.Document, error)
}
