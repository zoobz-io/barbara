package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/transformers"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ReleaseEntrySpec is one live path to write into a release: a key, the document
// it resolves to, and the version served. It is the cut input, decoupled from
// the stored ReleaseEntry (which also carries the surrogate and release ids).
type ReleaseEntrySpec struct {
	Key        string
	DocumentID string
	VersionID  string
}

// Releases is the data-access layer for releases — the immutable, append-only
// snapshots that are the only publish mechanism. It owns the release_entries
// table, and holds the app store (to lock the app row and move its pointer), the
// documents and versions stores (to snapshot the live tree), and the connection.
type Releases struct {
	*sum.Database[models.Release]
	db        *sqlx.DB
	entries   *sum.Database[models.ReleaseEntry]
	apps      *Apps
	documents *Documents
	versions  *Versions
	jobs      *Jobs
}

// NewReleases creates a releases store. It is the sole registrant of the
// release_entries table.
func NewReleases(db *sqlx.DB, renderer astql.Renderer, apps *Apps, documents *Documents, versions *Versions, jobs *Jobs) *Releases {
	return &Releases{
		Database:  sum.NewDatabase[models.Release](db, "releases", renderer),
		db:        db,
		entries:   sum.NewDatabase[models.ReleaseEntry](db, "release_entries", renderer),
		apps:      apps,
		documents: documents,
		versions:  versions,
		jobs:      jobs,
	}
}

// Cut snapshots the whole live tree: every non-deleted document with a head
// version becomes an entry at its key. The release row, its entries, and the
// app's moved pointer commit in one transaction.
func (s *Releases) Cut(ctx context.Context, appID string) (*models.Release, error) {
	return s.cutMode(ctx, appID, nil)
}

// CutWith cuts a release from an explicit entry set rather than a full-tree
// snapshot — the primitive the per-document publish sugar builds on.
func (s *Releases) CutWith(ctx context.Context, appID string, specs []ReleaseEntrySpec) (*models.Release, error) {
	return s.cutMode(ctx, appID, specs)
}

// Rollback cuts a NEW release copying an old release's entries forward. The
// pointer never moves backward; release numbers stay a straight line.
func (s *Releases) Rollback(ctx context.Context, appID, releaseID string) (*models.Release, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	createdBy, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}

	var release *models.Release
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		old, oldEntries, gerr := s.getTx(ctx, tx, appID, tenantID, releaseID)
		if gerr != nil {
			return gerr // ErrNotFound when the release is not the app's
		}
		_ = old
		specs := make([]ReleaseEntrySpec, len(oldEntries))
		for i, e := range oldEntries {
			specs[i] = ReleaseEntrySpec{Key: e.Key, DocumentID: e.DocumentID, VersionID: e.VersionID}
		}
		release, err = s.cut(ctx, tx, appID, tenantID, createdBy, specs)
		return err
	})
	if err != nil {
		return nil, err
	}
	events.Release.RolledBack.Emit(ctx, events.ReleaseRolledBackEvent{
		ReleaseID: release.ID, AppID: appID, TenantID: tenantID, Number: release.Number,
	})
	return release, nil
}

// List returns the app's releases, newest first, paginated.
func (s *Releases) List(ctx context.Context, appID string, limit, offset int) ([]*models.Release, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	releases, err := s.Query().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		OrderBy("number", "desc").
		Limit(limit).
		Offset(offset).
		Exec(ctx, map[string]any{"app_id": appID, "tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}
	return releases, nil
}

// Get returns a release with its entries, scoped to the app and tenant.
func (s *Releases) Get(ctx context.Context, appID, releaseID string) (*models.Release, []*models.ReleaseEntry, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.getTx(ctx, nil, appID, tenantID, releaseID)
}

// Contains reports whether a release carries the document — the current-release
// membership check the documents delete rules consult.
func (s *Releases) Contains(ctx context.Context, releaseID, documentID string) (bool, error) {
	n, err := s.entries.Count().
		Where("release_id", "=", "release_id").
		Where("document_id", "=", "document_id").
		Exec(ctx, map[string]any{"release_id": releaseID, "document_id": documentID})
	if err != nil {
		return false, fmt.Errorf("checking release membership: %w", err)
	}
	return n > 0, nil
}

// --- internals ---

// cutMode runs a full-tree (specs nil) or explicit-entry cut in one transaction.
func (s *Releases) cutMode(ctx context.Context, appID string, specs []ReleaseEntrySpec) (*models.Release, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	createdBy, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}

	var release *models.Release
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		if specs == nil {
			specs, err = s.snapshotHeads(ctx, tx, appID, tenantID)
			if err != nil {
				return err
			}
		}
		release, err = s.cut(ctx, tx, appID, tenantID, createdBy, specs)
		return err
	})
	if err != nil {
		return nil, err
	}
	events.Release.Cut.Emit(ctx, events.ReleaseCutEvent{
		ReleaseID: release.ID, AppID: appID, TenantID: tenantID, Number: release.Number,
	})
	return release, nil
}

// cut is the shared write: lock the app (existence + cut serialization), assign
// the next monotonic number, write the release row and its entries, and move the
// app's pointer forward. The app-row lock plus the unique (app_id, number) index
// keep the number monotonic under concurrent cuts.
func (s *Releases) cut(ctx context.Context, tx *sqlx.Tx, appID, tenantID, createdBy string, specs []ReleaseEntrySpec) (*models.Release, error) {
	app, err := s.apps.Select().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		ForUpdate().
		ExecTx(ctx, tx, map[string]any{"id": appID, "tenant_id": tenantID})
	if err != nil {
		return nil, err // ErrNotFound when the app is not the tenant's
	}
	prevReleaseID := app.CurrentReleaseID

	count, err := s.Count().
		Where("app_id", "=", "app_id").
		ExecTx(ctx, tx, map[string]any{"app_id": appID})
	if err != nil {
		return nil, fmt.Errorf("counting releases: %w", err)
	}

	now := time.Now()
	release, err := s.Insert().ExecTx(ctx, tx, &models.Release{
		AppID: appID, TenantID: tenantID, Number: int(count) + 1,
		CreatedBy: createdBy, CreatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("writing release: %w", err)
	}
	for _, spec := range specs {
		if _, err := s.entries.Insert().ExecTx(ctx, tx, &models.ReleaseEntry{
			ReleaseID: release.ID, Key: spec.Key,
			DocumentID: spec.DocumentID, VersionID: spec.VersionID,
		}); err != nil {
			return nil, fmt.Errorf("writing release entry %q: %w", spec.Key, err)
		}
	}
	if _, err := s.apps.Modify().
		Set("current_release_id", "current_release_id").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		ExecTx(ctx, tx, map[string]any{
			"current_release_id": release.ID, "updated_at": now,
			"id": appID, "tenant_id": tenantID,
		}); err != nil {
		return nil, fmt.Errorf("moving app pointer: %w", err)
	}
	// Project the diff against the previous release into the outbox, in the same
	// transaction — Postgres commits first, the index write follows via the jobs
	// pipeline (serving lags by seconds).
	if err := s.enqueueProjection(ctx, tx, tenantID, prevReleaseID, specs); err != nil {
		return nil, err
	}
	return release, nil
}

// enqueueProjection diffs the new entry set against the previous release and
// enqueues one index job per added/changed path and one delete job per removed
// path — keyed by document id, so a moved document (new key, same id) re-indexes
// rather than orphaning the old path. Commits with the cut; the pipeline lands
// the OpenSearch writes.
func (s *Releases) enqueueProjection(ctx context.Context, tx *sqlx.Tx, tenantID string, prevReleaseID *string, newSpecs []ReleaseEntrySpec) error {
	oldByDoc := map[string]ReleaseEntrySpec{}
	if prevReleaseID != nil {
		olds, err := s.entries.Query().
			Where("release_id", "=", "release_id").
			ExecTx(ctx, tx, map[string]any{"release_id": *prevReleaseID})
		if err != nil {
			return fmt.Errorf("loading previous entries: %w", err)
		}
		for _, e := range olds {
			oldByDoc[e.DocumentID] = ReleaseEntrySpec{Key: e.Key, DocumentID: e.DocumentID, VersionID: e.VersionID}
		}
	}

	newByDoc := make(map[string]bool, len(newSpecs))
	for _, spec := range newSpecs {
		newByDoc[spec.DocumentID] = true
		if old, ok := oldByDoc[spec.DocumentID]; ok && old.VersionID == spec.VersionID && old.Key == spec.Key {
			continue // unchanged path — nothing to re-index
		}
		payload, err := s.projectionPayload(ctx, tx, tenantID, spec)
		if err != nil {
			return err
		}
		if err := s.jobs.Enqueue(ctx, tx, newJob(tenantID, spec.DocumentID, models.JobIndex, payload)); err != nil {
			return err
		}
	}
	for docID := range oldByDoc {
		if !newByDoc[docID] {
			if err := s.jobs.Enqueue(ctx, tx, newJob(tenantID, docID, models.JobDelete, nil)); err != nil {
				return err
			}
		}
	}
	return nil
}

// projectionPayload loads the document and version behind an entry and marshals
// the OpenSearch projection (app_id and parent_path materialized).
func (s *Releases) projectionPayload(ctx context.Context, tx *sqlx.Tx, tenantID string, spec ReleaseEntrySpec) ([]byte, error) {
	doc, err := s.documents.Select().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		ExecTx(ctx, tx, map[string]any{"id": spec.DocumentID, "tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("loading document %s: %w", spec.DocumentID, err)
	}
	version, err := s.versions.Select().
		Where("id", "=", "id").
		ExecTx(ctx, tx, map[string]any{"id": spec.VersionID})
	if err != nil {
		return nil, fmt.Errorf("loading version %s: %w", spec.VersionID, err)
	}
	payload, err := json.Marshal(transformers.Projection(doc, version))
	if err != nil {
		return nil, fmt.Errorf("building projection: %w", err)
	}
	return payload, nil
}

// newJob builds a pending outbox job for an OpenSearch write.
func newJob(tenantID, documentID, operation string, payload []byte) *models.Job {
	now := time.Now()
	return &models.Job{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		DocumentID: documentID,
		Operation:  operation,
		Status:     models.JobPending,
		Payload:    models.JobPayload(payload),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// snapshotHeads builds the full-tree entry set: every non-deleted document in the
// app that has a head version, keyed by its materialized key.
func (s *Releases) snapshotHeads(ctx context.Context, tx *sqlx.Tx, appID, tenantID string) ([]ReleaseEntrySpec, error) {
	docs, err := s.documents.Query().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		WhereNull("deleted_at").
		OrderBy("key", "asc").
		ExecTx(ctx, tx, map[string]any{"app_id": appID, "tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing live documents: %w", err)
	}
	specs := make([]ReleaseEntrySpec, 0, len(docs))
	for _, doc := range docs {
		heads, err := s.versions.Query().
			Where("document_id", "=", "document_id").
			OrderBy("version_number", "desc").
			Limit(1).
			ExecTx(ctx, tx, map[string]any{"document_id": doc.ID})
		if err != nil {
			return nil, fmt.Errorf("loading head of %s: %w", doc.ID, err)
		}
		if len(heads) == 0 {
			continue // a document with no versions has nothing to publish
		}
		specs = append(specs, ReleaseEntrySpec{Key: doc.Key, DocumentID: doc.ID, VersionID: heads[0].ID})
	}
	return specs, nil
}

// getTx loads a release and its entries, scoped to app and tenant. tx may be nil
// for a non-transactional read.
func (s *Releases) getTx(ctx context.Context, tx *sqlx.Tx, appID, tenantID, releaseID string) (*models.Release, []*models.ReleaseEntry, error) {
	relQ := s.Select().
		Where("id", "=", "id").
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id")
	entryQ := s.entries.Query().
		Where("release_id", "=", "release_id").
		OrderBy("key", "asc")
	relArgs := map[string]any{"id": releaseID, "app_id": appID, "tenant_id": tenantID}
	entryArgs := map[string]any{"release_id": releaseID}

	var (
		release *models.Release
		entries []*models.ReleaseEntry
		err     error
	)
	if tx != nil {
		release, err = relQ.ExecTx(ctx, tx, relArgs)
	} else {
		release, err = relQ.Exec(ctx, relArgs)
	}
	if err != nil {
		return nil, nil, err // ErrNotFound when the release is not the app's
	}
	if tx != nil {
		entries, err = entryQ.ExecTx(ctx, tx, entryArgs)
	} else {
		entries, err = entryQ.Exec(ctx, entryArgs)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("loading release entries: %w", err)
	}
	return release, entries, nil
}

// inTx runs fn in a transaction, committing on success and rolling back on error.
func (s *Releases) inTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}
	return nil
}
