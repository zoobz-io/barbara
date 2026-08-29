package stores

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobz-io/barbara/database/transformers"
)

// zeroUUID is the keyset seed for a full reindex — smaller than every real
// app id, so the first page starts at the beginning.
const zeroUUID = "00000000-0000-0000-0000-000000000000"

// reindexBatch is how many apps a full reindex pages through at a time. Each
// app's current release is projected entry by entry; a release is
// markdown-scale, so its entries are loaded whole.
const reindexBatch = 100

// Reindex rebuilds the OpenSearch index from Postgres — the safety net behind
// the eventual-consistency serving model. The source of truth for "what is
// live" is each app's current release: it walks every app with a current
// release across all tenants and re-projects each entry, serving the ENTRY's
// key — a document moved after the cut serves its release-recorded path, not
// its authoring path. The projection build is the SAME one the release cut
// uses (transformers.Projection) — no divergent second code path. It returns
// the number of entries reindexed.
//
// Idempotent: each write upserts by document id, so re-running converges to
// one live entry per released document. It repopulates rather than reconciles
// — clearing entries for no-longer-released documents is the job of dropping
// the index (e.g. a mapping-change rebuild recreates it empty first). On an
// error it returns the count reindexed so far, so a re-run resumes safely.
//
// Tenant-agnostic operational machinery (cf. Search.SearchAll): the rebuild
// runs outside any tenant context and must see every tenant's releases. Not
// exposed on any tenant-facing surface.
func (s *Stores) Reindex(ctx context.Context) (int, error) {
	total := 0
	afterID := zeroUUID
	for {
		apps, err := s.Apps.Query().
			WhereNotNull("current_release_id").
			Where("id", ">", "after_id").
			OrderBy("id", "asc").
			Limit(reindexBatch).
			Exec(ctx, map[string]any{"after_id": afterID})
		if err != nil {
			return total, fmt.Errorf("enumerating apps with releases: %w", err)
		}
		if len(apps) == 0 {
			return total, nil
		}
		for _, app := range apps {
			entries, err := s.Releases.entries.Query().
				Where("release_id", "=", "release_id").
				OrderBy("key", "asc").
				Exec(ctx, map[string]any{"release_id": *app.CurrentReleaseID})
			if err != nil {
				return total, fmt.Errorf("loading entries of app %s: %w", app.ID, err)
			}
			for _, entry := range entries {
				doc, err := s.Documents.Select().
					Where("id", "=", "id").
					Where("tenant_id", "=", "tenant_id").
					Exec(ctx, map[string]any{"id": entry.DocumentID, "tenant_id": app.TenantID})
				if err != nil {
					return total, fmt.Errorf("loading document %s: %w", entry.DocumentID, err)
				}
				version, err := s.Versions.GetByID(ctx, entry.VersionID)
				if err != nil {
					return total, fmt.Errorf("loading version %s: %w", entry.VersionID, err)
				}
				payload, err := json.Marshal(transformers.Projection(doc, version, entry.Key))
				if err != nil {
					return total, fmt.Errorf("building projection for document %s: %w", doc.ID, err)
				}
				if err := s.Search.Index(ctx, doc.ID, payload); err != nil {
					return total, fmt.Errorf("indexing document %s: %w", doc.ID, err)
				}
				total++
			}
			afterID = app.ID
		}
		if len(apps) < reindexBatch {
			return total, nil
		}
	}
}
