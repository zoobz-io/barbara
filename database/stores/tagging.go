package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/transformers"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// AddTag adds a tag to a document; adding a tag it already carries is a no-op.
// Returns the updated document.
func (s *Stores) AddTag(ctx context.Context, documentID, tag string) (*models.Document, error) {
	return s.changeTags(ctx, documentID, func(tags []string) ([]string, bool) {
		return transformers.AddTag(tags, tag)
	}, func(ctx context.Context, doc *models.Document) {
		events.Document.TagAdded.Emit(ctx, events.DocumentTagAddedEvent{
			DocumentID: doc.ID, TenantID: doc.TenantID, Tag: tag,
		})
	})
}

// RemoveTag removes a tag from a document; removing a tag it doesn't carry is a
// no-op. Returns the updated document.
func (s *Stores) RemoveTag(ctx context.Context, documentID, tag string) (*models.Document, error) {
	return s.changeTags(ctx, documentID, func(tags []string) ([]string, bool) {
		return transformers.RemoveTag(tags, tag)
	}, func(ctx context.Context, doc *models.Document) {
		events.Document.TagRemoved.Emit(ctx, events.DocumentTagRemovedEvent{
			DocumentID: doc.ID, TenantID: doc.TenantID, Tag: tag,
		})
	})
}

// changeTags applies mutate to a document's tags atomically. It locks the
// document row (FOR UPDATE) so concurrent tag changes serialize rather than
// losing each other's updates, then writes the new tags. If the document is
// published, it enqueues a reprojection in the SAME transaction — the projection
// carries tags, so the OpenSearch entry must update — WITHOUT moving the
// published pointer: a tag change is metadata, not a new publish. A no-op
// mutation (tag already present / already absent) writes nothing and enqueues
// nothing. Returns ErrNotFound if the document is absent for the tenant.
func (s *Stores) changeTags(ctx context.Context, documentID string, mutate func([]string) ([]string, bool), emit func(context.Context, *models.Document)) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var result *models.Document
	var changed bool
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		doc, lockErr := s.Documents.Select().
			Where("id", "=", "id").
			Where("tenant_id", "=", "tenant_id").
			ForUpdate().
			ExecTx(ctx, tx, map[string]any{"id": documentID, "tenant_id": tenantID})
		if lockErr != nil {
			return lockErr // soy.ErrNotFound when the document is absent for the tenant
		}

		newTags, didChange := mutate([]string(doc.Tags))
		if !didChange {
			result = doc
			return nil
		}
		changed = true

		result, err = s.setTags(ctx, tx, documentID, tenantID, newTags)
		if err != nil {
			return err
		}
		// Reproject only if the document is live — carried by the app's current
		// release — using the release-recorded version and path, not the head.
		entry, entryErr := s.Releases.CurrentEntryFor(ctx, result.AppID, documentID)
		if entryErr != nil {
			return entryErr
		}
		if entry == nil {
			return nil // not in the current release: Postgres only, nothing to reproject
		}
		return s.enqueueReprojection(ctx, tx, tenantID, result, entry)
	})
	if err != nil {
		return nil, err
	}
	// Post-commit: a no-op tag change (already present / already absent) emits
	// nothing, mirroring its Postgres/outbox no-op.
	if changed {
		emit(ctx, result)
	}
	return result, nil
}

// setTags writes a document's tags (and bumps updated_at) within tx, returning
// the updated row.
func (s *Stores) setTags(ctx context.Context, tx *sqlx.Tx, documentID, tenantID string, tags []string) (*models.Document, error) {
	return s.Documents.Modify().
		Set("tags", "tags").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		ExecTx(ctx, tx, map[string]any{
			"tags":       pq.StringArray(tags),
			"updated_at": time.Now(),
			"id":         documentID,
			"tenant_id":  tenantID,
		})
}

// enqueueReprojection enqueues an index job rebuilding the OpenSearch entry from
// the document and the version the current release serves — used when a live
// document's metadata (tags) changes, so the index entry must update without
// cutting a new release. The entry's key and version keep the projection aligned
// with what is live, even if the head has moved on.
func (s *Stores) enqueueReprojection(ctx context.Context, tx *sqlx.Tx, tenantID string, doc *models.Document, entry *models.ReleaseEntry) error {
	version, err := s.Versions.Get(ctx, entry.VersionID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(transformers.Projection(doc, version, entry.Key))
	if err != nil {
		return fmt.Errorf("building projection: %w", err)
	}
	return s.Jobs.Enqueue(ctx, tx, newJob(tenantID, doc.ID, models.JobIndex, payload))
}
