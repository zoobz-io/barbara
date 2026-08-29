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
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrNotFound is returned when a document does not exist for the tenant.
var ErrNotFound = soy.ErrNotFound

// ErrDocumentPublished is returned when deleting a document that is still
// published — it must be unpublished first.
var ErrDocumentPublished = errors.New("document is published; unpublish before deleting")

// Documents is the data-access layer for the logical document. Every method is
// scoped to the tenant carried in the request context; the tree operations are
// additionally scoped to an app. It holds the connection (to run tree mutations
// in a transaction), the versions store (head enrichment), the collections
// store (materialized-key resolution and the cross-table sibling check), and
// read handles onto apps and release_entries (the delete rules).
type Documents struct {
	*sum.Database[models.Document]
	db          *sqlx.DB
	versions    *Versions    // head-version enrichment; wired in New
	collections *Collections // tree path + sibling namespace; wired in New (breaks the cycle)
	apps        *Apps        // current-release pointer; wired in New
	releases    *Releases    // current-release membership (delete rules); wired in New
}

// NewDocuments creates a documents store.
func NewDocuments(db *sqlx.DB, renderer astql.Renderer) *Documents {
	return &Documents{
		Database: sum.NewDatabase[models.Document](db, "documents", renderer),
		db:       db,
	}
}

// Create inserts a new document placed in the tree: under collectionID (nil =
// app root) in the app, with the given name. The key is materialized from the
// collection's path and the name. The name must be unique among sibling
// collections and documents; a collision returns ErrCollectionNameTaken.
// Validates that the collection (or, at the root, the app) belongs to the tenant.
func (s *Documents) Create(ctx context.Context, appID string, collectionID *string, name string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if serr := s.collections.requireParentScope(ctx, appID, tenantID, collectionID); serr != nil {
		return nil, serr
	}

	var created *models.Document
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		path, perr := s.collections.pathOf(ctx, tx, appID, tenantID, collectionID)
		if perr != nil {
			return perr
		}
		// The key unique index guards document-vs-document; this guards the
		// cross-table half (a collection sibling with the same name).
		taken, cerr := s.siblingCollectionExists(ctx, tx, appID, tenantID, collectionID, name)
		if cerr != nil {
			return cerr
		}
		if taken {
			return ErrCollectionNameTaken
		}
		now := time.Now()
		created, err = s.Insert().ExecTx(ctx, tx, &models.Document{
			TenantID:     tenantID,
			AppID:        &appID,
			CollectionID: collectionID,
			Name:         &name,
			Key:          joinPath(path, name),
			Tags:         pq.StringArray{},
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrCollectionNameTaken
			}
			return fmt.Errorf("creating document: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	events.Document.Created.Emit(ctx, events.DocumentCreatedEvent{
		DocumentID: created.ID, TenantID: tenantID, Key: created.Key,
	})
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

// ListByTag returns the tenant's documents carrying the given tag, oldest
// first, paginated. The filter is a Postgres array-containment test on the tags
// column (tags @> ARRAY[tag]).
func (s *Documents) ListByTag(ctx context.Context, tag string, limit, offset int) ([]*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	docs, err := s.Query().
		Where("tenant_id", "=", "tenant_id").
		Where("tags", "@>", "tag").
		OrderBy("created_at", "asc").
		Limit(limit).
		Offset(offset).
		Exec(ctx, map[string]any{"tenant_id": tenantID, "tag": pq.StringArray{tag}})
	if err != nil {
		return nil, fmt.Errorf("listing documents by tag: %w", err)
	}
	return docs, nil
}

// Status derives a document's lifecycle status from the app's current release:
// draft when the release does not carry it, published when the release carries
// its head version, and published-with-newer-draft when the release carries an
// older version than the head (a newer draft is waiting). An unplaced document
// (no app) is always a draft.
func (s *Documents) Status(ctx context.Context, doc *models.Document) (string, error) {
	if doc.AppID == nil {
		return models.StatusDraft, nil
	}
	entry, err := s.releases.CurrentEntryFor(ctx, *doc.AppID, doc.ID)
	if err != nil {
		return "", err
	}
	if entry == nil {
		return models.StatusDraft, nil
	}
	head, err := s.versions.Head(ctx, doc.ID)
	if err != nil {
		return "", err
	}
	// The head is always the latest version, so a different id in the release
	// entry means the live version is behind the head — a newer draft exists.
	if head == nil || head.ID == entry.VersionID {
		return models.StatusPublished, nil
	}
	return models.StatusPublishedWithNewerDraft, nil
}

// Statuses derives the status of several documents, returned by document id. It
// loads each app's current-release entries once, then compares each document's
// head against its entry.
func (s *Documents) Statuses(ctx context.Context, docs []*models.Document) (map[string]string, error) {
	out := make(map[string]string, len(docs))
	entryByDoc := map[string]string{} // document id -> live version id in the current release
	loadedApps := map[string]bool{}
	for _, doc := range docs {
		if doc.AppID == nil {
			out[doc.ID] = models.StatusDraft
			continue
		}
		if !loadedApps[*doc.AppID] {
			entries, err := s.releases.CurrentEntries(ctx, *doc.AppID)
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				entryByDoc[e.DocumentID] = e.VersionID
			}
			loadedApps[*doc.AppID] = true
		}
		liveVersion, live := entryByDoc[doc.ID]
		if !live {
			out[doc.ID] = models.StatusDraft
			continue
		}
		head, err := s.versions.Head(ctx, doc.ID)
		if err != nil {
			return nil, err
		}
		if head == nil || head.ID == liveVersion {
			out[doc.ID] = models.StatusPublished
		} else {
			out[doc.ID] = models.StatusPublishedWithNewerDraft
		}
	}
	return out, nil
}

// GetWithHead retrieves a document together with its head (latest) version — the
// read behind opening a document for editing in one call. Head is nil when
// the document has no versions yet (an empty document, not a 404).
func (s *Documents) GetWithHead(ctx context.Context, id string) (*models.DocumentHead, error) {
	doc, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	head, err := s.versions.Head(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.DocumentHead{Document: doc, Head: head}, nil
}

// Move reparents a document under newCollectionID (nil = app root) and/or renames
// it, rewriting the materialized key in the same transaction. A same-collection
// move with a new name is the rename path. The name must be unique among sibling
// collections and documents at the destination. Returns ErrNotFound if the
// document does not exist for the tenant.
func (s *Documents) Move(ctx context.Context, appID, id string, newCollectionID *string, newName string) (*models.Document, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if serr := s.collections.requireParentScope(ctx, appID, tenantID, newCollectionID); serr != nil {
		return nil, serr
	}

	var moved *models.Document
	err = s.inTx(ctx, func(tx *sqlx.Tx) error {
		// Lock the document row, serializing concurrent moves of it (and
		// confirming it exists for the tenant in the app).
		if _, lerr := s.Select().
			Where("id", "=", "id").
			Where("app_id", "=", "app_id").
			Where("tenant_id", "=", "tenant_id").
			ForUpdate().
			ExecTx(ctx, tx, map[string]any{"id": id, "app_id": appID, "tenant_id": tenantID}); lerr != nil {
			return lerr
		}
		path, perr := s.collections.pathOf(ctx, tx, appID, tenantID, newCollectionID)
		if perr != nil {
			return perr
		}
		taken, cerr := s.siblingCollectionExists(ctx, tx, appID, tenantID, newCollectionID, newName)
		if cerr != nil {
			return cerr
		}
		if taken {
			return ErrCollectionNameTaken
		}
		moved, err = s.Modify().
			Set("collection_id", "collection_id").
			Set("name", "name").
			Set("key", "key").
			Set("updated_at", "updated_at").
			Where("id", "=", "id").
			Where("tenant_id", "=", "tenant_id").
			ExecTx(ctx, tx, map[string]any{
				"collection_id": newCollectionID,
				"name":          newName,
				"key":           joinPath(path, newName),
				"updated_at":    time.Now(),
				"id":            id,
				"tenant_id":     tenantID,
			})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrCollectionNameTaken
			}
			return fmt.Errorf("moving document: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	events.Document.Moved.Emit(ctx, events.DocumentMovedEvent{
		DocumentID: id, TenantID: tenantID, CollectionID: newCollectionID, Key: moved.Key,
	})
	return moved, nil
}

// Delete removes a document that is absent from the app's current release. A
// document carried by the current release (or still holding a published pointer,
// during the pre-release-publishing transition) is refused with
// ErrDocumentPublished. Otherwise the document is hard-deleted (versions cascade)
// unless a historical release references it — the release_entries RESTRICT
// foreign key surfaces that as a violation, and the document is soft-deleted
// instead: deleted_at set, key freed (the partial unique index), versions kept.
func (s *Documents) Delete(ctx context.Context, id string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	doc, err := s.Get(ctx, id)
	if err != nil {
		return err // ErrNotFound when absent for the tenant
	}
	live, err := s.inCurrentRelease(ctx, doc)
	if err != nil {
		return err
	}
	if live {
		return ErrDocumentPublished
	}

	n, err := s.Remove().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"id": id, "tenant_id": tenantID})
	if err != nil {
		if isForeignKeyViolation(err) {
			// A historical release references it: keep the row and its versions,
			// free the key. deleted_at drops it out of the partial key index.
			if _, serr := s.Modify().
				Set("deleted_at", "deleted_at").
				Set("updated_at", "updated_at").
				Where("id", "=", "id").
				Where("tenant_id", "=", "tenant_id").
				Exec(ctx, map[string]any{
					"deleted_at": time.Now(), "updated_at": time.Now(),
					"id": id, "tenant_id": tenantID,
				}); serr != nil {
				return fmt.Errorf("soft-deleting document: %w", serr)
			}
			events.Document.Deleted.Emit(ctx, events.DocumentDeletedEvent{DocumentID: id, TenantID: tenantID})
			return nil
		}
		return fmt.Errorf("deleting document: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	events.Document.Deleted.Emit(ctx, events.DocumentDeletedEvent{DocumentID: id, TenantID: tenantID})
	return nil
}

// inCurrentRelease reports whether the document is carried by its app's current
// release. An unplaced document (no app) or an app with no current release is
// not in any release.
func (s *Documents) inCurrentRelease(ctx context.Context, doc *models.Document) (bool, error) {
	if doc.AppID == nil {
		return false, nil
	}
	app, err := s.apps.Get(ctx, *doc.AppID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("loading app: %w", err)
	}
	if app.CurrentReleaseID == nil {
		return false, nil
	}
	return s.releases.Contains(ctx, *app.CurrentReleaseID, doc.ID)
}

// siblingCollectionExists reports whether a collection already occupies
// (app, parent, name) — the cross-table half of the sibling namespace, checked
// when placing or moving a document.
func (s *Documents) siblingCollectionExists(ctx context.Context, tx *sqlx.Tx, appID, tenantID string, parentID *string, name string) (bool, error) {
	q := s.collections.Count().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Where("name", "=", "name")
	args := map[string]any{"app_id": appID, "tenant_id": tenantID, "name": name}
	if parentID != nil {
		q = q.Where("parent_id", "=", "parent_id")
		args["parent_id"] = *parentID
	} else {
		q = q.WhereNull("parent_id")
	}
	n, err := q.ExecTx(ctx, tx, args)
	if err != nil {
		return false, fmt.Errorf("checking sibling collections: %w", err)
	}
	return n > 0, nil
}

// inTx runs fn in a transaction, committing on success and rolling back on error.
func (s *Documents) inTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
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
