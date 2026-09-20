package stores

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/zoobz-io/barbara/database/models"
)

// RebuildReleases recomputes every release's change rows and counts from its
// entries and its predecessor's, across all apps and tenants — the entries
// are the source of truth; the counts and release_changes are a projection of
// adjacent manifests. Run it to backfill releases that predate the metadata
// columns, or to repair rows that drifted. Tenant-agnostic operational
// tooling, like the reindex: it walks apps by keyset and rebuilds each app's
// history in number order. Returns the number of releases rebuilt.
func (s *Stores) RebuildReleases(ctx context.Context) (int, error) {
	total := 0
	afterID := zeroUUID
	for {
		apps, err := s.Apps.Query().
			Where("id", ">", "after_id").
			OrderBy("id", "asc").
			Limit(reindexBatch).
			Exec(ctx, map[string]any{"after_id": afterID})
		if err != nil {
			return total, fmt.Errorf("enumerating apps: %w", err)
		}
		if len(apps) == 0 {
			return total, nil
		}
		for _, app := range apps {
			n, err := s.Releases.rebuildApp(ctx, app.ID)
			if err != nil {
				return total, fmt.Errorf("rebuilding releases of app %s: %w", app.ID, err)
			}
			total += n
		}
		if len(apps) < reindexBatch {
			return total, nil // a short page is the last one
		}
		afterID = apps[len(apps)-1].ID
	}
}

// rebuildApp walks an app's releases oldest first, diffing each against the
// one before it, and replaces each release's change rows and counts in its
// own transaction. The kind, label, and provenance columns are facts of the
// cut, not derivable from entries, and are left alone.
func (s *Releases) rebuildApp(ctx context.Context, appID string) (int, error) {
	releases, err := s.Query().
		Where("app_id", "=", "app_id").
		OrderBy("number", "asc").
		Exec(ctx, map[string]any{"app_id": appID})
	if err != nil {
		return 0, fmt.Errorf("listing releases: %w", err)
	}
	var prev []*models.ReleaseEntry
	for _, release := range releases {
		entries, err := s.entriesTx(ctx, nil, release.ID)
		if err != nil {
			return 0, err
		}
		specs := make([]ReleaseEntrySpec, len(entries))
		for i, e := range entries {
			specs[i] = ReleaseEntrySpec{Key: e.Key, DocumentID: e.DocumentID, VersionID: e.VersionID, VersionNumber: e.VersionNumber}
		}
		diff := diffEntries(prev, specs)
		if err := s.inTx(ctx, func(tx *sqlx.Tx) error {
			return s.replaceChanges(ctx, tx, release.ID, len(specs), diff)
		}); err != nil {
			return 0, err
		}
		prev = entries
	}
	return len(releases), nil
}

// replaceChanges swaps a release's change rows for the diff and writes its
// counts, in the caller's transaction.
func (s *Releases) replaceChanges(ctx context.Context, tx *sqlx.Tx, releaseID string, entryCount int, diff releaseDiff) error {
	if _, err := s.changes.Remove().
		Where("release_id", "=", "release_id").
		ExecTx(ctx, tx, map[string]any{"release_id": releaseID}); err != nil {
		return fmt.Errorf("clearing changes of release %s: %w", releaseID, err)
	}
	if err := s.writeChanges(ctx, tx, releaseID, diff); err != nil {
		return err
	}
	if _, err := s.Modify().
		Set("entry_count", "entry_count").
		Set("added", "added").
		Set("changed", "changed").
		Set("removed", "removed").
		Set("moved", "moved").
		Where("id", "=", "id").
		ExecTx(ctx, tx, map[string]any{
			"entry_count": entryCount, "added": diff.added, "changed": diff.changed,
			"removed": diff.removed, "moved": diff.moved, "id": releaseID,
		}); err != nil {
		return fmt.Errorf("writing counts of release %s: %w", releaseID, err)
	}
	return nil
}
