package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
// it resolves to, and the version served (id and number). It is the cut input,
// decoupled from the stored ReleaseEntry (which also carries the surrogate and
// release ids).
type ReleaseEntrySpec struct {
	Key           string
	DocumentID    string
	VersionID     string
	VersionNumber int
}

// CutMeta is what a cut records about itself beyond its entries: the kind of
// operation, an optional label, and — for a rollback or a single-document
// publish — the release copied or the document concerned.
type CutMeta struct {
	SourceReleaseID   *string
	SubjectDocumentID *string
	Kind              string
	Label             string
}

// Releases is the data-access layer for releases — the immutable, append-only
// snapshots that are the only publish mechanism. It owns the release_entries
// and release_changes tables, and holds the app store (to lock the app row and
// move its pointer), the documents and versions stores (to snapshot the live
// tree), and the connection.
type Releases struct {
	*sum.Database[models.Release]
	db        *sqlx.DB
	entries   *sum.Database[models.ReleaseEntry]
	changes   *sum.Database[models.ReleaseChange]
	apps      *Apps
	documents *Documents
	versions  *Versions
	jobs      *Jobs
}

// NewReleases creates a releases store. It is the sole registrant of the
// release_entries and release_changes tables.
func NewReleases(db *sqlx.DB, renderer astql.Renderer, apps *Apps, documents *Documents, versions *Versions, jobs *Jobs) *Releases {
	return &Releases{
		Database:  sum.NewDatabase[models.Release](db, "releases", renderer),
		db:        db,
		entries:   sum.NewDatabase[models.ReleaseEntry](db, "release_entries", renderer),
		changes:   sum.NewDatabase[models.ReleaseChange](db, "release_changes", renderer),
		apps:      apps,
		documents: documents,
		versions:  versions,
		jobs:      jobs,
	}
}

// Cut snapshots the whole live tree: every non-deleted document with a head
// version becomes an entry at its key. The release row, its entries, its
// changes against the previous release, and the app's moved pointer commit in
// one transaction. label is optional ("" for none).
func (s *Releases) Cut(ctx context.Context, appID, label string) (*models.Release, error) {
	return s.cutMode(ctx, appID, nil, CutMeta{Kind: models.ReleaseKindCut, Label: label})
}

// CutWith cuts a release from an explicit entry set rather than a full-tree
// snapshot — the primitive the per-document publish sugar builds on. The meta
// records what the cut was.
func (s *Releases) CutWith(ctx context.Context, appID string, specs []ReleaseEntrySpec, meta CutMeta) (*models.Release, error) {
	if specs == nil {
		specs = []ReleaseEntrySpec{} // an explicit empty set, not a full-tree cut
	}
	return s.cutMode(ctx, appID, specs, meta)
}

// Rollback cuts a NEW release copying an old release's entries forward. The
// pointer never moves backward; release numbers stay a straight line. The new
// release records the old one as its source. label is optional.
func (s *Releases) Rollback(ctx context.Context, appID, releaseID, label string) (*models.Release, error) {
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
		specs := make([]ReleaseEntrySpec, len(oldEntries))
		for i, e := range oldEntries {
			specs[i] = ReleaseEntrySpec{Key: e.Key, DocumentID: e.DocumentID, VersionID: e.VersionID, VersionNumber: e.VersionNumber}
		}
		meta := CutMeta{Kind: models.ReleaseKindRollback, Label: label, SourceReleaseID: &old.ID}
		release, err = s.cut(ctx, tx, appID, tenantID, createdBy, specs, meta)
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

// Total returns how many releases the app has — the list's true total. (Count
// is the embedded builder, which the apps delete guard uses directly.)
func (s *Releases) Total(ctx context.Context, appID string) (int64, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return 0, err
	}
	n, err := s.Database.Count().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"app_id": appID, "tenant_id": tenantID})
	if err != nil {
		return 0, fmt.Errorf("counting releases: %w", err)
	}
	return int64(n), nil
}

// Get returns a release with its entries, scoped to the app and tenant.
func (s *Releases) Get(ctx context.Context, appID, releaseID string) (*models.Release, []*models.ReleaseEntry, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.getTx(ctx, nil, appID, tenantID, releaseID)
}

// Changes returns a release with its changes against the previous release,
// by key, scoped to the app and tenant.
func (s *Releases) Changes(ctx context.Context, appID, releaseID string) (*models.Release, []*models.ReleaseChange, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, nil, err
	}
	release, err := s.releaseTx(ctx, nil, appID, tenantID, releaseID)
	if err != nil {
		return nil, nil, err
	}
	changes, err := s.changes.Query().
		Where("release_id", "=", "release_id").
		OrderBy("key", "asc").
		Exec(ctx, map[string]any{"release_id": releaseID})
	if err != nil {
		return nil, nil, fmt.Errorf("loading release changes: %w", err)
	}
	return release, changes, nil
}

// CurrentEntries returns the entries of the app's current release, or an empty
// slice when the app has no current release. The publish sugar builds the next
// release's entry set from these.
func (s *Releases) CurrentEntries(ctx context.Context, appID string) ([]*models.ReleaseEntry, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	app, err := s.apps.Get(ctx, appID)
	if err != nil {
		return nil, err // ErrNotFound when the app is not the tenant's
	}
	if app.CurrentReleaseID == nil {
		return nil, nil
	}
	_, entries, err := s.getTx(ctx, nil, appID, tenantID, *app.CurrentReleaseID)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// CurrentEntryFor returns the document's entry in the app's current release, or
// nil when the document is not live — the read behind derived status.
func (s *Releases) CurrentEntryFor(ctx context.Context, appID, documentID string) (*models.ReleaseEntry, error) {
	app, err := s.apps.Get(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.CurrentReleaseID == nil {
		return nil, nil
	}
	entries, err := s.entries.Query().
		Where("release_id", "=", "release_id").
		Where("document_id", "=", "document_id").
		Exec(ctx, map[string]any{"release_id": *app.CurrentReleaseID, "document_id": documentID})
	if err != nil {
		return nil, fmt.Errorf("loading current entry: %w", err)
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return entries[0], nil
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

// --- the diff ---

// releaseDiff is a release's entry set compared with the previous release's:
// the change rows (release id unset until written) and their counts. Keyed by
// document, so a document at a new key is a move, not a removal plus an
// addition. Deterministic in its input order: the new entries first, then the
// documents that went away, in the previous release's order.
type releaseDiff struct {
	changes []*models.ReleaseChange
	added   int
	changed int
	removed int
	moved   int
}

// diffEntries compares the next entry set with the previous release's entries.
func diffEntries(prev []*models.ReleaseEntry, next []ReleaseEntrySpec) releaseDiff {
	var d releaseDiff
	oldByDoc := make(map[string]*models.ReleaseEntry, len(prev))
	for _, e := range prev {
		oldByDoc[e.DocumentID] = e
	}
	newByDoc := make(map[string]bool, len(next))
	for _, spec := range next {
		newByDoc[spec.DocumentID] = true
		versionID, versionNumber := spec.VersionID, spec.VersionNumber
		change := &models.ReleaseChange{
			Key: spec.Key, DocumentID: spec.DocumentID,
			VersionID: &versionID, VersionNumber: &versionNumber,
		}
		old, ok := oldByDoc[spec.DocumentID]
		switch {
		case !ok:
			change.Change = models.ChangeAdded
			d.added++
		case old.Key != spec.Key:
			prevKey, prevID, prevNumber := old.Key, old.VersionID, old.VersionNumber
			change.Change = models.ChangeMoved
			change.PrevKey, change.PrevVersionID, change.PrevVersionNumber = &prevKey, &prevID, &prevNumber
			d.moved++
		case old.VersionID != spec.VersionID:
			prevID, prevNumber := old.VersionID, old.VersionNumber
			change.Change = models.ChangeChanged
			change.PrevVersionID, change.PrevVersionNumber = &prevID, &prevNumber
			d.changed++
		default:
			continue // unchanged path — no row
		}
		d.changes = append(d.changes, change)
	}
	for _, e := range prev {
		if newByDoc[e.DocumentID] {
			continue
		}
		prevID, prevNumber := e.VersionID, e.VersionNumber
		d.changes = append(d.changes, &models.ReleaseChange{
			Key: e.Key, DocumentID: e.DocumentID, Change: models.ChangeRemoved,
			PrevVersionID: &prevID, PrevVersionNumber: &prevNumber,
		})
		d.removed++
	}
	return d
}

// --- internals ---

// cutMode runs a full-tree (specs nil) or explicit-entry cut in one transaction.
func (s *Releases) cutMode(ctx context.Context, appID string, specs []ReleaseEntrySpec, meta CutMeta) (*models.Release, error) {
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
		release, err = s.cut(ctx, tx, appID, tenantID, createdBy, specs, meta)
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
// the next monotonic number, diff the entry set against the current release,
// write the release row with its counts, its entries, and its change rows, and
// move the app's pointer forward. The app-row lock plus the unique (app_id,
// number) index keep the number monotonic under concurrent cuts.
func (s *Releases) cut(ctx context.Context, tx *sqlx.Tx, appID, tenantID, createdBy string, specs []ReleaseEntrySpec, meta CutMeta) (*models.Release, error) {
	app, err := s.apps.Select().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		ForUpdate().
		ExecTx(ctx, tx, map[string]any{"id": appID, "tenant_id": tenantID})
	if err != nil {
		return nil, err // ErrNotFound when the app is not the tenant's
	}

	count, err := s.Database.Count().
		Where("app_id", "=", "app_id").
		ExecTx(ctx, tx, map[string]any{"app_id": appID})
	if err != nil {
		return nil, fmt.Errorf("counting releases: %w", err)
	}

	var prev []*models.ReleaseEntry
	if app.CurrentReleaseID != nil {
		prev, err = s.entriesTx(ctx, tx, *app.CurrentReleaseID)
		if err != nil {
			return nil, err
		}
	}
	diff := diffEntries(prev, specs)

	now := time.Now()
	release, err := s.Insert().ExecTx(ctx, tx, &models.Release{
		AppID: appID, TenantID: tenantID, Number: int(count) + 1,
		CreatedBy: createdBy, CreatedAt: now,
		Kind: meta.Kind, Label: optionalText(meta.Label),
		SourceReleaseID: meta.SourceReleaseID, SubjectDocumentID: meta.SubjectDocumentID,
		EntryCount: len(specs),
		Added:      diff.added, Changed: diff.changed, Removed: diff.removed, Moved: diff.moved,
	})
	if err != nil {
		return nil, fmt.Errorf("writing release: %w", err)
	}
	for _, spec := range specs {
		if _, err := s.entries.Insert().ExecTx(ctx, tx, &models.ReleaseEntry{
			ReleaseID: release.ID, Key: spec.Key,
			DocumentID: spec.DocumentID, VersionID: spec.VersionID, VersionNumber: spec.VersionNumber,
		}); err != nil {
			return nil, fmt.Errorf("writing release entry %q: %w", spec.Key, err)
		}
	}
	if err := s.writeChanges(ctx, tx, release.ID, diff); err != nil {
		return nil, err
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
	// Project the diff into the outbox, in the same transaction — Postgres
	// commits first, the index write follows via the jobs pipeline (serving
	// lags by seconds).
	if err := s.enqueueProjection(ctx, tx, tenantID, diff); err != nil {
		return nil, err
	}
	return release, nil
}

// writeChanges inserts a diff's change rows for the release.
func (s *Releases) writeChanges(ctx context.Context, tx *sqlx.Tx, releaseID string, diff releaseDiff) error {
	for _, c := range diff.changes {
		row := c.Clone()
		row.ReleaseID = releaseID
		if _, err := s.changes.Insert().ExecTx(ctx, tx, &row); err != nil {
			return fmt.Errorf("writing release change %q: %w", c.Key, err)
		}
	}
	return nil
}

// enqueueProjection turns the diff into outbox jobs: one index job per added,
// changed, or moved path and one delete job per removed path — keyed by
// document id, so a moved document (new key, same id) re-indexes rather than
// orphaning the old path. Commits with the cut; the pipeline lands the
// OpenSearch writes.
func (s *Releases) enqueueProjection(ctx context.Context, tx *sqlx.Tx, tenantID string, diff releaseDiff) error {
	for _, c := range diff.changes {
		if c.Change == models.ChangeRemoved {
			if err := s.jobs.Enqueue(ctx, tx, newJob(tenantID, c.DocumentID, models.JobDelete, nil)); err != nil {
				return err
			}
			continue
		}
		spec := ReleaseEntrySpec{Key: c.Key, DocumentID: c.DocumentID, VersionID: *c.VersionID}
		payload, err := s.projectionPayload(ctx, tx, tenantID, spec)
		if err != nil {
			return err
		}
		if err := s.jobs.Enqueue(ctx, tx, newJob(tenantID, c.DocumentID, models.JobIndex, payload)); err != nil {
			return err
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
	payload, err := json.Marshal(transformers.Projection(doc, version, spec.Key))
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

// optionalText trims a label and returns nil for an empty one, so the column
// holds NULL rather than "".
func optionalText(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
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
		specs = append(specs, ReleaseEntrySpec{
			Key: doc.Key, DocumentID: doc.ID,
			VersionID: heads[0].ID, VersionNumber: heads[0].VersionNumber,
		})
	}
	return specs, nil
}

// releaseTx loads a release scoped to app and tenant. tx may be nil for a
// non-transactional read.
func (s *Releases) releaseTx(ctx context.Context, tx *sqlx.Tx, appID, tenantID, releaseID string) (*models.Release, error) {
	q := s.Select().
		Where("id", "=", "id").
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id")
	args := map[string]any{"id": releaseID, "app_id": appID, "tenant_id": tenantID}
	if tx != nil {
		return q.ExecTx(ctx, tx, args) // ErrNotFound when the release is not the app's
	}
	return q.Exec(ctx, args)
}

// entriesTx loads a release's entries by key. tx may be nil.
func (s *Releases) entriesTx(ctx context.Context, tx *sqlx.Tx, releaseID string) ([]*models.ReleaseEntry, error) {
	q := s.entries.Query().
		Where("release_id", "=", "release_id").
		OrderBy("key", "asc")
	args := map[string]any{"release_id": releaseID}
	var (
		entries []*models.ReleaseEntry
		err     error
	)
	if tx != nil {
		entries, err = q.ExecTx(ctx, tx, args)
	} else {
		entries, err = q.Exec(ctx, args)
	}
	if err != nil {
		return nil, fmt.Errorf("loading release entries: %w", err)
	}
	return entries, nil
}

// getTx loads a release and its entries, scoped to app and tenant. tx may be nil
// for a non-transactional read.
func (s *Releases) getTx(ctx context.Context, tx *sqlx.Tx, appID, tenantID, releaseID string) (*models.Release, []*models.ReleaseEntry, error) {
	release, err := s.releaseTx(ctx, tx, appID, tenantID, releaseID)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.entriesTx(ctx, tx, releaseID)
	if err != nil {
		return nil, nil, err
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
