package stores

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrNoUser is returned when a write reaches the store without an acting user
// in context — every version records who created it.
var ErrNoUser = errors.New("no acting user in context")

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

// Save appends a new version of a document's content, allocating the next
// version_number for that document.
//
// It runs in a transaction that first locks the parent document row
// (FOR UPDATE), so concurrent saves for the same document serialize: each sees
// a stable count and inserts count+1, and both persist with distinct monotonic
// numbers — the race the domain guarantees against. Saves for different
// documents don't contend (row-level lock). Returns ErrNotFound if the document
// does not exist for the tenant.
func (s *Versions) Save(ctx context.Context, documentID, content string) (*models.Version, error) {
	tenantID, err := tenant(ctx)
	if err != nil {
		return nil, err
	}
	createdBy := auth.UserFromContext(ctx)
	if createdBy == "" {
		return nil, ErrNoUser
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
	return created, nil
}

// List returns a document's versions, newest first, scoped to the tenant.
func (s *Versions) List(ctx context.Context, documentID string, limit, offset int) ([]*models.Version, error) {
	tenantID, err := tenant(ctx)
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
	tenantID, err := tenant(ctx)
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
