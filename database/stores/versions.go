package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// Versions is the data-access layer for immutable document versions.
type Versions struct {
	*sum.Database[models.Version]
	db        *sqlx.DB
	documents *Documents
}

// NewVersions creates a versions store. It holds the connection (to run Save in
// a transaction) and the documents store (to lock the parent on save).
func NewVersions(db *sqlx.DB, renderer astql.Renderer, documents *Documents) *Versions {
	return &Versions{
		Database:  sum.NewDatabase[models.Version](db, "versions", renderer),
		db:        db,
		documents: documents,
	}
}

// VersionConflictError is returned by Save when the caller's baseVersion is no
// longer the document's head — a concurrent save moved it. CurrentHead is the
// head the client should refetch and rebase onto before retrying.
type VersionConflictError struct {
	CurrentHead int
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("stale base_version; current head is %d", e.CurrentHead)
}

// Save appends a new version of a document's content, allocating the next
// version_number — but only if baseVersion is still the document's head
// (optimistic concurrency). baseVersion is the version number the caller edited
// from; 0 for the first version of an empty document.
//
// It runs in a transaction that first locks the parent document row
// (FOR UPDATE), so concurrent saves for the same document serialize. Inside the
// lock it compares baseVersion against the current head (the version count): if
// they differ, a concurrent save has moved the head, and it returns a
// *VersionConflictError carrying that head — the compare and the insert are one
// atomic step, so of two racing saves from the same base exactly one wins and
// the other conflicts. Saves for different documents don't contend (row-level
// lock). Returns ErrNotFound if the document does not exist for the tenant.
func (s *Versions) Save(ctx context.Context, documentID, content string, baseVersion int) (*models.Version, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	createdBy, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning save tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after a successful commit

	scope := map[string]any{"document_id": documentID, "tenant_id": tenantID}

	// Lock the parent document row, serializing saves for this document (and
	// confirming it exists in the tenant).
	if _, lockErr := s.documents.Select().
		Where("id", "=", "document_id").
		Where("tenant_id", "=", "tenant_id").
		ForUpdate().
		ExecTx(ctx, tx, scope); lockErr != nil {
		return nil, lockErr // soy.ErrNotFound when the document is absent
	}

	count, err := s.Count().
		Where("document_id", "=", "document_id").
		Where("tenant_id", "=", "tenant_id").
		ExecTx(ctx, tx, scope)
	if err != nil {
		return nil, fmt.Errorf("counting versions: %w", err)
	}

	// Optimistic concurrency: the head is the current version count (versions are
	// numbered 1..count with no gaps). Reject unless the caller edited from it.
	if baseVersion != int(count) {
		return nil, &VersionConflictError{CurrentHead: int(count)}
	}

	v := &models.Version{
		DocumentID:    documentID,
		TenantID:      tenantID,
		VersionNumber: int(count) + 1,
		Content:       content,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
	created, err := s.Insert().ExecTx(ctx, tx, v)
	if err != nil {
		return nil, fmt.Errorf("saving version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing version: %w", err)
	}
	events.Version.Saved.Emit(ctx, events.VersionSavedEvent{
		VersionID: created.ID, DocumentID: documentID, TenantID: tenantID, VersionNumber: created.VersionNumber,
	})
	return created, nil
}

// GetByID retrieves a version by primary key, WITHOUT tenant scoping —
// operational machinery for the full reindex, which runs outside any tenant
// context and loads each published version to rebuild its projection. The
// version id is a globally-unique primary key, so this is unambiguous.
// Tenant-facing callers use Get, which scopes to the request's tenant.
func (s *Versions) GetByID(ctx context.Context, id string) (*models.Version, error) {
	v, err := s.Select().
		Where("id", "=", "id").
		Exec(ctx, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	return v, nil
}

// List returns a document's versions, newest first, scoped to the tenant.
func (s *Versions) List(ctx context.Context, documentID string, limit, offset int) ([]*models.Version, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	versions, err := s.Query().
		Where("document_id", "=", "document_id").
		Where("tenant_id", "=", "tenant_id").
		OrderBy("version_number", "desc").
		Limit(limit).
		Offset(offset).
		Exec(ctx, map[string]any{"document_id": documentID, "tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}
	return versions, nil
}

// Get retrieves a version by ID, scoped to the tenant.
func (s *Versions) Get(ctx context.Context, id string) (*models.Version, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.Select().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"id": id, "tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Head returns a document's latest version, scoped to the tenant, or nil when the
// document has no versions yet (an empty document, not an error).
func (s *Versions) Head(ctx context.Context, documentID string) (*models.Version, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	versions, err := s.Query().
		Where("document_id", "=", "document_id").
		Where("tenant_id", "=", "tenant_id").
		OrderBy("version_number", "desc").
		Limit(1).
		Exec(ctx, map[string]any{"document_id": documentID, "tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("loading head version: %w", err)
	}
	if len(versions) == 0 {
		return nil, nil
	}
	return versions[0], nil
}

