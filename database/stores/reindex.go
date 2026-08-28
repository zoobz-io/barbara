package stores

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobz-io/barbara/database/transformers"
)

// zeroUUID is the keyset seed for a full reindex — smaller than every real
// document id, so the first page starts at the beginning.
const zeroUUID = "00000000-0000-0000-0000-000000000000"

// reindexBatch is how many published documents a full reindex loads and
// re-projects per page.
const reindexBatch = 100

// Reindex rebuilds the OpenSearch index from Postgres — the safety net behind
// the eventual-consistency serving model. It walks every published document
// across all tenants and writes each merged projection through the search
// store's write side, using the SAME projection build as publish
// (transformers.Projection) — no divergent second code path. It returns the
// number of documents reindexed.
//
// Idempotent: each write upserts by document id, so re-running converges to one
// live entry per published document. It repopulates rather than reconciles —
// clearing entries for no-longer-published documents is the job of dropping the
// index (e.g. a mapping-change rebuild recreates it empty first). On an error it
// returns the count reindexed so far, so a re-run resumes safely.
func (s *Stores) Reindex(ctx context.Context) (int, error) {
	total := 0
	afterID := zeroUUID
	for {
		docs, err := s.Documents.ListPublishedAfter(ctx, afterID, reindexBatch)
		if err != nil {
			return total, fmt.Errorf("enumerating published documents: %w", err)
		}
		if len(docs) == 0 {
			return total, nil
		}
		for _, doc := range docs {
			version, err := s.Versions.GetByID(ctx, *doc.PublishedVersionID)
			if err != nil {
				return total, fmt.Errorf("loading published version for document %s: %w", doc.ID, err)
			}
			payload, err := json.Marshal(transformers.Projection(doc, version))
			if err != nil {
				return total, fmt.Errorf("building projection for document %s: %w", doc.ID, err)
			}
			if err := s.Search.Index(ctx, doc.ID, payload); err != nil {
				return total, fmt.Errorf("indexing document %s: %w", doc.ID, err)
			}
			total++
			afterID = doc.ID
		}
		if len(docs) < reindexBatch {
			return total, nil
		}
	}
}
