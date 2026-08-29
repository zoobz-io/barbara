package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Tagging is the authoring view of a document's organizational tags. Adding or
// removing a tag on a published document re-projects it into OpenSearch without
// moving the published pointer; on a draft it touches only Postgres. The
// aggregate store implements it, since the published case spans documents and
// the jobs outbox atomically. Every method is tenant-scoped via the request
// context.
type Tagging interface {
	// AddTag adds a tag to a document (idempotent).
	AddTag(ctx context.Context, documentID, tag string) (*models.Document, error)
	// RemoveTag removes a tag from a document (idempotent).
	RemoveTag(ctx context.Context, documentID, tag string) (*models.Document, error)
}
