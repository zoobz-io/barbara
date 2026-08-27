package stores

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/soy"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrNotFound is returned when a document does not exist for the tenant.
var ErrNotFound = soy.ErrNotFound

// ErrDocumentPublished is returned when deleting a document that is still
// published — it must be unpublished first.
var ErrDocumentPublished = errors.New("document is published; unpublish before deleting")

// Documents is the data-access layer for the logical document. Every method is
// scoped to the tenant carried in the request context.
type Documents struct {
	*sum.Database[models.Document]
}

// NewDocuments creates a documents store.
func NewDocuments(db *sqlx.DB, renderer astql.Renderer) *Documents {
	return &Documents{Database: sum.NewDatabase[models.Document](db, "documents", renderer)}
}

// Create inserts a new document with the given key for the request's tenant.
// The key must be unique per tenant; a duplicate returns an error.
func (s *Documents) Create(ctx context.Context, key string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	// Timestamps are set here rather than left to the column default: the insert
	// carries every non-primary-key field, so a zero time.Time would otherwise
	// override the DB default. The id is DB-generated (primary key, omitted).
	now := time.Now()
	doc := &models.Document{
		TenantID:  tenantID,
		Key:       key,
		Tags:      pq.StringArray{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := s.Insert().Exec(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("creating document: %w", err)
	}
	return created, nil
}

// Get retrieves a document by ID, scoped to the request's tenant.
func (s *Documents) Get(ctx context.Context, id string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.Select().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"id": id, "tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// List returns the tenant's documents, oldest first, paginated.
func (s *Documents) List(ctx context.Context, limit, offset int) ([]*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	docs, err := s.Query().
		Where("tenant_id", "=", "tenant_id").
		OrderBy("created_at", "asc").
		Limit(limit).
		Offset(offset).
		Exec(ctx, map[string]any{"tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing documents: %w", err)
	}
	return docs, nil
}

// Rename changes a document's key, freeing the old one (the old key 404s
// afterward). The new key must be unique per tenant. Returns ErrNotFound if the
// document does not exist for the tenant.
func (s *Documents) Rename(ctx context.Context, id, newKey string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.Modify().
		Set("key", "key").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{
			"key":        newKey,
			"updated_at": time.Now(),
			"id":         id,
			"tenant_id":  tenantID,
		})
	if err != nil {
		return nil, fmt.Errorf("renaming document: %w", err)
	}
	return doc, nil
}

// Delete removes an unpublished document (and cascades to its versions). A
// published document is refused with ErrDocumentPublished; a missing one with
// ErrNotFound.
func (s *Documents) Delete(ctx context.Context, id string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	n, err := s.Remove().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		WhereNull("published_version_id").
		Exec(ctx, map[string]any{"id": id, "tenant_id": tenantID})
	if err != nil {
		return fmt.Errorf("deleting document: %w", err)
	}
	if n == 0 {
		// Nothing deleted: either it doesn't exist, or it's published.
		if _, getErr := s.Get(ctx, id); errors.Is(getErr, ErrNotFound) {
			return ErrNotFound
		}
		return ErrDocumentPublished
	}
	return nil
}

