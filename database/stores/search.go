package stores

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
)

// documentIndex is the OpenSearch index holding the published-document
// projection — one live entry per published document, keyed by document ID.
const documentIndex = "documents"

// Search is the serving-store access layer over the DocumentIndex projection.
// The jobs pipeline writes through it (Index/Delete below) and the site-facing
// surface reads through it. It embeds sum.Search for the typed query/index
// primitives; the write side here takes the job's raw JSON payload so the
// pipeline stays decoupled from the projection type.
type Search struct {
	*sum.Search[models.DocumentIndex]
}

// NewSearch creates the search store against the documents index.
func NewSearch(provider grub.SearchProvider) *Search {
	return &Search{Search: sum.NewSearch[models.DocumentIndex](provider, documentIndex)}
}

// Index upserts a document projection by document ID. payload is the JSONB
// projection carried by the job; it is decoded into a DocumentIndex and written
// as the OpenSearch document. Satisfies jobs.IndexWriter.
func (s *Search) Index(ctx context.Context, documentID string, payload []byte) error {
	var doc models.DocumentIndex
	if err := json.Unmarshal(payload, &doc); err != nil {
		return fmt.Errorf("decoding projection for %s: %w", documentID, err)
	}
	if err := s.Search.Index(ctx, documentID, &doc); err != nil {
		return fmt.Errorf("indexing document %s: %w", documentID, err)
	}
	return nil
}

// Delete removes a document's projection by document ID. Satisfies
// jobs.IndexWriter.
func (s *Search) Delete(ctx context.Context, documentID string) error {
	if err := s.Search.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("deleting document %s: %w", documentID, err)
	}
	return nil
}
