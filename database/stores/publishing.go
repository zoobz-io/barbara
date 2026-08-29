package stores

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/transformers"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrVersionMismatch is returned when publishing a version that belongs to a
// different document.
var ErrVersionMismatch = errors.New("version does not belong to the document")

// Publish points a document at the given version and projects the merged
// document into OpenSearch. The pointer move commits to Postgres first; the
// OpenSearch write follows via the jobs pipeline and retries until it lands, so
// serving lags authoring by seconds. Returns the updated document.
func (s *Stores) Publish(ctx context.Context, documentID, versionID string) (*models.Document, error) {
	updated, err := s.publishVersion(ctx, documentID, versionID)
	if err != nil {
		return nil, err
	}
	events.Document.Published.Emit(ctx, events.DocumentPublishedEvent{
		DocumentID: documentID, TenantID: updated.TenantID, VersionID: versionID,
	})
	return updated, nil
}

// Rollback republishes an older version: the same pointer-move-and-reindex as
// Publish, nothing copied. Mechanically identical; named for intent.
func (s *Stores) Rollback(ctx context.Context, documentID, versionID string) (*models.Document, error) {
	updated, err := s.publishVersion(ctx, documentID, versionID)
	if err != nil {
		return nil, err
	}
	events.Document.RolledBack.Emit(ctx, events.DocumentRolledBackEvent{
		DocumentID: documentID, TenantID: updated.TenantID, VersionID: versionID,
	})
	return updated, nil
}

// publishVersion is the shared publish/rollback core: validate the version
// belongs to the document, build the projection, then atomically move the
// published pointer and enqueue the index write.
func (s *Stores) publishVersion(ctx context.Context, documentID, versionID string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	version, err := s.Versions.Get(ctx, versionID)
	if err != nil {
		return nil, err // ErrNotFound when the version is absent for the tenant
	}
	if version.DocumentID != documentID {
		return nil, ErrVersionMismatch
	}
	doc, err := s.Documents.Get(ctx, documentID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(transformers.Projection(doc, version))
	if err != nil {
		return nil, fmt.Errorf("building projection: %w", err)
	}

	var updated *models.Document
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		updated, err = s.setPublishedVersion(ctx, tx, documentID, tenantID, versionID)
		if err != nil {
			return err
		}
		return s.Jobs.Enqueue(ctx, tx, newJob(tenantID, documentID, models.JobIndex, payload))
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Unpublish clears a document's published pointer and enqueues removal of its
// OpenSearch entry. Returns the updated document.
func (s *Stores) Unpublish(ctx context.Context, documentID string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var updated *models.Document
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		updated, err = s.setPublishedVersion(ctx, tx, documentID, tenantID, nil)
		if err != nil {
			return err
		}
		return s.Jobs.Enqueue(ctx, tx, newJob(tenantID, documentID, models.JobDelete, nil))
	})
	if err != nil {
		return nil, err
	}
	events.Document.Unpublished.Emit(ctx, events.DocumentUnpublishedEvent{
		DocumentID: documentID, TenantID: tenantID,
	})
	return updated, nil
}

// setPublishedVersion moves (or clears, when versionID is nil) a document's
// published pointer within tx, returning the updated row. ErrNotFound when the
// document is absent for the tenant.
func (s *Stores) setPublishedVersion(ctx context.Context, tx *sqlx.Tx, documentID, tenantID string, versionID any) (*models.Document, error) {
	return s.Documents.Modify().
		Set("published_version_id", "published_version_id").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		ExecTx(ctx, tx, map[string]any{
			"published_version_id": versionID,
			"updated_at":           time.Now(),
			"id":                   documentID,
			"tenant_id":            tenantID,
		})
}

// inTx runs fn in a transaction, committing on success and rolling back on error.
func (s *Stores) inTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
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

