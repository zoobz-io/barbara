// Package contracts defines the interfaces the public-API surface depends on:
// the read interface over the search store and the authoring interfaces over
// the shared stores. Each exposes only what its surface needs.
package contracts

import (
	"context"

	"github.com/zoobz-io/barbara/database/models"
)

// Reads is the site-facing view of published documents, served from OpenSearch
// exclusively. Every method is tenant-scoped via the request context — the
// search store refuses to run without a tenant. Admin-only cross-tenant search
// (SearchAll) is deliberately absent here.
type Reads interface {
	// GetPublishedByKey returns the published document with the given key.
	GetPublishedByKey(ctx context.Context, key string) (*models.DocumentIndex, error)
	// Enumerate lists published documents, optionally filtered by tag,
	// returning the page and the total match count.
	Enumerate(ctx context.Context, tag string, limit, offset int) ([]models.DocumentIndex, int64, error)
	// Search runs a full-text search over published content, returning the page
	// and the total match count.
	Search(ctx context.Context, query string, limit, offset int) ([]models.DocumentIndex, int64, error)
}
