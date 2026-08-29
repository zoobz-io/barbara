package stores

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/lucene"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/internal/auth"
)

// documentIndex is the OpenSearch index holding the published-document
// projection — one live entry per published document, keyed by document ID.
const documentIndex = "documents"

// Deterministic sort orders for the paginated reads. Every listing ends with a
// unique tiebreaker (document_id, a keyword) so a page is a total order:
// offset paging never skips or repeats a document, even as the index merges
// segments. A listing (filter context, no relevance score) reads oldest-first,
// matching Documents.List; a full-text search reads by relevance first, then the
// tiebreaker. Without this, ties fall back to internal Lucene doc order, which
// is not stable across paginated requests.
var (
	sortByCreatedAt = lucene.SortField{Field: "created_at", Order: "asc"}
	sortByScore     = lucene.SortField{Field: "_score", Order: "desc"}
	sortByDocID     = lucene.SortField{Field: "document_id", Order: "asc"}
)

// Search is the serving-store access layer over the DocumentIndex projection.
// The jobs pipeline writes through it (Index/Delete) and the site-facing surface
// reads through it (GetPublishedByKey/Enumerate/Search). The write side takes
// the job's raw JSON payload so the pipeline stays decoupled from the projection
// type; the read side builds typed lucene queries, tenant-scoped.
type Search struct {
	index *sum.Search[models.DocumentIndex]
	qb    *lucene.Builder[models.DocumentIndex]
}

// NewSearch creates the search store against the documents index.
func NewSearch(provider grub.SearchProvider) *Search {
	return &Search{
		index: sum.NewSearch[models.DocumentIndex](provider, documentIndex),
		qb:    lucene.New[models.DocumentIndex](),
	}
}

// Index upserts a document projection by document ID. payload is the JSONB
// projection carried by the job; it is decoded into a DocumentIndex and written
// as the OpenSearch document. Satisfies jobs.IndexWriter.
func (s *Search) Index(ctx context.Context, documentID string, payload []byte) error {
	var doc models.DocumentIndex
	if err := json.Unmarshal(payload, &doc); err != nil {
		return fmt.Errorf("decoding projection for %s: %w", documentID, err)
	}
	if err := s.index.Index(ctx, documentID, &doc); err != nil {
		return fmt.Errorf("indexing document %s: %w", documentID, err)
	}
	return nil
}

// Delete removes a document's projection by document ID. Satisfies
// jobs.IndexWriter.
func (s *Search) Delete(ctx context.Context, documentID string) error {
	if err := s.index.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("deleting document %s: %w", documentID, err)
	}
	return nil
}

// GetPublishedByKey returns the published document with the given key, scoped to
// the request's tenant. ErrNotFound when no published document has that key.
func (s *Search) GetPublishedByKey(ctx context.Context, key string) (*models.DocumentIndex, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	q := s.qb.Bool().Filter(s.qb.Term("tenant_id", tenantID), s.qb.Term("key", key))
	res, err := s.index.Execute(ctx, lucene.NewSearch().Query(q).Size(1))
	if err != nil {
		return nil, fmt.Errorf("looking up %q: %w", key, err)
	}
	if len(res.Hits) == 0 {
		return nil, ErrNotFound
	}
	doc := res.Hits[0].Content
	return &doc, nil
}

// Enumerate lists published documents for the request's tenant, optionally
// filtered by tag. Returns the page and the total match count.
func (s *Search) Enumerate(ctx context.Context, tag string, limit, offset int) ([]models.DocumentIndex, int64, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, 0, err
	}
	filters := []lucene.Query{s.qb.Term("tenant_id", tenantID)}
	if tag != "" {
		filters = append(filters, s.qb.Term("tags", tag))
	}
	return s.run(ctx, s.qb.Bool().Filter(filters...), limit, offset, sortByCreatedAt, sortByDocID)
}

// Search runs a full-text search over published content for the request's
// tenant. Returns the page and the total match count.
func (s *Search) Search(ctx context.Context, query string, limit, offset int) ([]models.DocumentIndex, int64, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := s.qb.Bool().
		Filter(s.qb.Term("tenant_id", tenantID)).
		Must(s.qb.MultiMatch(query, "content"))
	return s.run(ctx, q, limit, offset, sortByScore, sortByDocID)
}

// SearchAll runs a full-text search across all tenants — admin use only, not
// exposed on the site-facing surface.
func (s *Search) SearchAll(ctx context.Context, query string, limit, offset int) ([]models.DocumentIndex, int64, error) {
	q := s.qb.Bool().Must(s.qb.MultiMatch(query, "content"))
	return s.run(ctx, q, limit, offset, sortByScore, sortByDocID)
}

// run executes a query and returns the page of projections plus the total. Every
// caller passes a sort ending in a unique tiebreaker so paging is stable.
func (s *Search) run(ctx context.Context, q lucene.Query, limit, offset int, sort ...lucene.SortField) ([]models.DocumentIndex, int64, error) {
	res, err := s.index.Execute(ctx, lucene.NewSearch().Query(q).Size(limit).From(offset).Sort(sort...))
	if err != nil {
		return nil, 0, fmt.Errorf("executing search: %w", err)
	}
	docs := make([]models.DocumentIndex, len(res.Hits))
	for i, hit := range res.Hits {
		docs[i] = hit.Content
	}
	return docs, res.Total, nil
}
