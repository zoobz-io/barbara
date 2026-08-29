package stores

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
)

// ErrVersionMismatch is returned when publishing a version that belongs to a
// different document.
var ErrVersionMismatch = errors.New("version does not belong to the document")

// Publish makes a version the document's live version by cutting a release of
// the app's current tree with this document's entry set to it. Publish state
// lives in releases now, so this is sugar over the release primitive; the
// #63 projection lands the OpenSearch write. Returns the document.
func (s *Stores) Publish(ctx context.Context, documentID, versionID string) (*models.Document, error) {
	doc, err := s.cutForDocument(ctx, documentID, &versionID)
	if err != nil {
		return nil, err
	}
	events.Document.Published.Emit(ctx, events.DocumentPublishedEvent{
		DocumentID: documentID, TenantID: doc.TenantID, VersionID: versionID,
	})
	return doc, nil
}

// Rollback republishes an older version — mechanically a publish of that
// version; named for intent.
func (s *Stores) Rollback(ctx context.Context, documentID, versionID string) (*models.Document, error) {
	doc, err := s.cutForDocument(ctx, documentID, &versionID)
	if err != nil {
		return nil, err
	}
	events.Document.RolledBack.Emit(ctx, events.DocumentRolledBackEvent{
		DocumentID: documentID, TenantID: doc.TenantID, VersionID: versionID,
	})
	return doc, nil
}

// Unpublish drops a document from the live site by cutting a release of the
// current tree without its path. Returns the document.
func (s *Stores) Unpublish(ctx context.Context, documentID string) (*models.Document, error) {
	doc, err := s.cutForDocument(ctx, documentID, nil)
	if err != nil {
		return nil, err
	}
	events.Document.Unpublished.Emit(ctx, events.DocumentUnpublishedEvent{
		DocumentID: documentID, TenantID: doc.TenantID,
	})
	return doc, nil
}

// cutForDocument cuts a release of the app's current entries with this
// document's entry set to versionID (publish/rollback) or removed (unpublish,
// versionID nil). The single-document publish sugar the plan calls for: one
// mechanism underneath, no second publish system. Returns the document.
func (s *Stores) cutForDocument(ctx context.Context, documentID string, versionID *string) (*models.Document, error) {
	doc, err := s.Documents.Get(ctx, documentID)
	if err != nil {
		return nil, err // ErrNotFound when the document is absent for the tenant
	}
	appID := doc.AppID

	if versionID != nil {
		version, verr := s.Versions.Get(ctx, *versionID)
		if verr != nil {
			return nil, verr // ErrNotFound when the version is absent
		}
		if version.DocumentID != documentID {
			return nil, ErrVersionMismatch
		}
	}

	current, err := s.Releases.CurrentEntries(ctx, appID)
	if err != nil {
		return nil, err
	}
	// Carry every current path except this document's, then re-add it at the new
	// version when publishing (omit it when unpublishing).
	specs := make([]ReleaseEntrySpec, 0, len(current)+1)
	for _, e := range current {
		if e.DocumentID == documentID {
			continue
		}
		specs = append(specs, ReleaseEntrySpec{Key: e.Key, DocumentID: e.DocumentID, VersionID: e.VersionID})
	}
	if versionID != nil {
		specs = append(specs, ReleaseEntrySpec{Key: doc.Key, DocumentID: documentID, VersionID: *versionID})
	}

	if _, err := s.Releases.CutWith(ctx, appID, specs); err != nil {
		return nil, err
	}
	return doc, nil
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

