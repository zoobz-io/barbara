package stores

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrCollectionNotEmpty is returned when deleting a collection that still holds
// subcollections or documents — delete is rmdir, not rm -rf.
var ErrCollectionNotEmpty = errors.New("collection is not empty")

// ErrCollectionNameTaken is returned when a create/rename/move would collide
// with a sibling — another collection (the unique index) or a document sharing
// the parent and name (the cross-table half, checked in the transaction).
var ErrCollectionNameTaken = errors.New("a collection or document with that name already exists in the parent")

// ErrCollectionCycle is returned when moving a collection into itself or one of
// its own descendants.
var ErrCollectionCycle = errors.New("cannot move a collection into itself or a descendant")

// Collections is the data-access layer for the folder tree. Every method is
// scoped to the tenant (context) and the app (argument). It holds the documents
// store to rewrite descendant keys and check the cross-table sibling namespace,
// the apps store to validate app ownership at the tree root, and the connection
// to run the multi-table mutations (rename/move) in one transaction.
type Collections struct {
	*sum.Database[models.Collection]
	db        *sqlx.DB
	documents *Documents
	apps      *Apps
}

// NewCollections creates a collections store.
func NewCollections(db *sqlx.DB, renderer astql.Renderer, documents *Documents, apps *Apps) *Collections {
	return &Collections{
		Database:  sum.NewDatabase[models.Collection](db, "collections", renderer),
		db:        db,
		documents: documents,
		apps:      apps,
	}
}

// Create makes a collection under parentID (nil = app root) in the app. The name
// must be unique among sibling collections and documents; a collision returns
// ErrCollectionNameTaken. Validates that the parent (or, at the root, the app)
// belongs to the tenant.
func (s *Collections) Create(ctx context.Context, appID string, parentID *string, name string) (*models.Collection, error) {
	tenantID, terr := auth.RequireTenant(ctx)
	if terr != nil {
		return nil, terr
	}
	if serr := s.requireParentScope(ctx, appID, tenantID, parentID); serr != nil {
		return nil, serr
	}

	var created *models.Collection
	err := s.inTx(ctx, func(tx *sqlx.Tx) error {
		// The unique index guards collection-vs-collection; this guards the
		// cross-table half (a document sibling with the same name).
		taken, err := s.siblingDocumentExists(ctx, tx, appID, tenantID, parentID, name)
		if err != nil {
			return err
		}
		if taken {
			return ErrCollectionNameTaken
		}
		now := time.Now()
		created, err = s.Insert().ExecTx(ctx, tx, &models.Collection{
			TenantID: tenantID, AppID: appID, ParentID: parentID, Name: name,
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrCollectionNameTaken
			}
			return fmt.Errorf("creating collection: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	events.Collection.Created.Emit(ctx, events.CollectionCreatedEvent{
		CollectionID: created.ID, TenantID: tenantID, AppID: appID, Name: created.Name,
	})
	return created, nil
}

// Get retrieves a collection by ID, scoped to the tenant and app.
func (s *Collections) Get(ctx context.Context, appID, id string) (*models.Collection, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	return s.scoped(ctx, appID, tenantID, id)
}

// ListContents returns a collection's direct subcollections and documents in one
// call (collectionID nil = the app root). Subcollections are ordered by name,
// documents by key; soft-deleted documents are excluded.
func (s *Collections) ListContents(ctx context.Context, appID string, collectionID *string) (*models.CollectionContents, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.requireParentScope(ctx, appID, tenantID, collectionID); err != nil {
		return nil, err
	}

	subs, err := s.childCollections(ctx, nil, appID, tenantID, collectionID)
	if err != nil {
		return nil, fmt.Errorf("listing subcollections: %w", err)
	}
	docs, err := s.childDocuments(ctx, nil, appID, tenantID, collectionID)
	if err != nil {
		return nil, fmt.Errorf("listing documents: %w", err)
	}
	return &models.CollectionContents{Subcollections: subs, Documents: docs}, nil
}

// Rename changes a collection's name and rewrites every descendant document key
// in the same transaction. The new name must be unique among siblings.
func (s *Collections) Rename(ctx context.Context, appID, id, newName string) (*models.Collection, error) {
	tenantID, terr := auth.RequireTenant(ctx)
	if terr != nil {
		return nil, terr
	}

	var updated *models.Collection
	renamed := false
	err := s.inTx(ctx, func(tx *sqlx.Tx) error {
		current, err := s.lock(ctx, tx, appID, tenantID, id)
		if err != nil {
			return err
		}
		if current.Name == newName {
			updated = current // no-op rename: nothing to rewrite, nothing to emit
			return nil
		}
		taken, err := s.siblingDocumentExists(ctx, tx, appID, tenantID, current.ParentID, newName)
		if err != nil {
			return err
		}
		if taken {
			return ErrCollectionNameTaken
		}
		parentPath, err := s.pathOf(ctx, tx, appID, tenantID, current.ParentID)
		if err != nil {
			return err
		}
		oldPath := joinPath(parentPath, current.Name)
		newPath := joinPath(parentPath, newName)

		updated, err = s.Modify().
			Set("name", "name").
			Set("updated_at", "updated_at").
			Where("id", "=", "id").
			Where("app_id", "=", "app_id").
			Where("tenant_id", "=", "tenant_id").
			ExecTx(ctx, tx, map[string]any{
				"name": newName, "updated_at": time.Now(),
				"id": id, "app_id": appID, "tenant_id": tenantID,
			})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrCollectionNameTaken
			}
			return fmt.Errorf("renaming collection: %w", err)
		}
		renamed = true
		return s.rewriteDescendantKeys(ctx, tx, appID, tenantID, oldPath, newPath)
	})
	if err != nil {
		return nil, err
	}
	if renamed {
		events.Collection.Renamed.Emit(ctx, events.CollectionRenamedEvent{
			CollectionID: id, TenantID: tenantID, AppID: appID, Name: newName,
		})
	}
	return updated, nil
}

// Move reparents a collection under newParentID (nil = app root) and rewrites
// every descendant document key in the same transaction. Refuses a move into the
// collection itself or a descendant (ErrCollectionCycle); the new name must be
// unique among siblings at the destination.
func (s *Collections) Move(ctx context.Context, appID, id string, newParentID *string) (*models.Collection, error) {
	tenantID, terr := auth.RequireTenant(ctx)
	if terr != nil {
		return nil, terr
	}

	var updated *models.Collection
	moved := false
	err := s.inTx(ctx, func(tx *sqlx.Tx) error {
		current, err := s.lock(ctx, tx, appID, tenantID, id)
		if err != nil {
			return err
		}
		if samePtr(current.ParentID, newParentID) {
			updated = current // already there
			return nil
		}
		if err = s.checkMoveTarget(ctx, tx, appID, tenantID, id, newParentID); err != nil {
			return err
		}
		taken, err := s.siblingDocumentExists(ctx, tx, appID, tenantID, newParentID, current.Name)
		if err != nil {
			return err
		}
		if taken {
			return ErrCollectionNameTaken
		}
		oldParentPath, err := s.pathOf(ctx, tx, appID, tenantID, current.ParentID)
		if err != nil {
			return err
		}
		newParentPath, err := s.pathOf(ctx, tx, appID, tenantID, newParentID)
		if err != nil {
			return err
		}
		oldPath := joinPath(oldParentPath, current.Name)
		newPath := joinPath(newParentPath, current.Name)

		updated, err = s.Modify().
			Set("parent_id", "parent_id").
			Set("updated_at", "updated_at").
			Where("id", "=", "id").
			Where("app_id", "=", "app_id").
			Where("tenant_id", "=", "tenant_id").
			ExecTx(ctx, tx, map[string]any{
				"parent_id": newParentID, "updated_at": time.Now(),
				"id": id, "app_id": appID, "tenant_id": tenantID,
			})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrCollectionNameTaken
			}
			return fmt.Errorf("moving collection: %w", err)
		}
		moved = true
		return s.rewriteDescendantKeys(ctx, tx, appID, tenantID, oldPath, newPath)
	})
	if err != nil {
		return nil, err
	}
	if moved {
		events.Collection.Moved.Emit(ctx, events.CollectionMovedEvent{
			CollectionID: id, TenantID: tenantID, AppID: appID, ParentID: newParentID,
		})
	}
	return updated, nil
}

// Delete removes an empty collection. A collection holding any subcollection or
// document is refused with ErrCollectionNotEmpty; a missing one with ErrNotFound.
// The emptiness check is the friendly guard; the child foreign keys are the
// backstop if a child is added between the check and the delete.
func (s *Collections) Delete(ctx context.Context, appID, id string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(tx *sqlx.Tx) error {
		if _, err := s.lock(ctx, tx, appID, tenantID, id); err != nil {
			return err
		}
		empty, err := s.isEmpty(ctx, tx, appID, tenantID, id)
		if err != nil {
			return err
		}
		if !empty {
			return ErrCollectionNotEmpty
		}
		n, err := s.Remove().
			Where("id", "=", "id").
			Where("app_id", "=", "app_id").
			Where("tenant_id", "=", "tenant_id").
			ExecTx(ctx, tx, map[string]any{"id": id, "app_id": appID, "tenant_id": tenantID})
		if err != nil {
			if isForeignKeyViolation(err) {
				return ErrCollectionNotEmpty // a child was added between the check and here
			}
			return fmt.Errorf("deleting collection: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}
		events.Collection.Deleted.Emit(ctx, events.CollectionDeletedEvent{
			CollectionID: id, TenantID: tenantID, AppID: appID,
		})
		return nil
	})
}

// --- internals ---

// scoped loads a collection by (id, app, tenant). ErrNotFound when absent.
func (s *Collections) scoped(ctx context.Context, appID, tenantID, id string) (*models.Collection, error) {
	return s.Select().
		Where("id", "=", "id").
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"id": id, "app_id": appID, "tenant_id": tenantID})
}

// lock loads a collection by (id, app, tenant) FOR UPDATE, serializing concurrent
// tree edits of the same collection. ErrNotFound when absent.
func (s *Collections) lock(ctx context.Context, tx *sqlx.Tx, appID, tenantID, id string) (*models.Collection, error) {
	return s.Select().
		Where("id", "=", "id").
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		ForUpdate().
		ExecTx(ctx, tx, map[string]any{"id": id, "app_id": appID, "tenant_id": tenantID})
}

// requireParentScope validates that a create/list target's container belongs to
// the tenant: the parent collection when nested, else the app itself at the root.
func (s *Collections) requireParentScope(ctx context.Context, appID, tenantID string, parentID *string) error {
	if parentID != nil {
		// ErrNotFound scopes out another tenant's or another app's collection.
		_, err := s.scoped(ctx, appID, tenantID, *parentID)
		return err
	}
	if _, err := s.apps.Get(ctx, appID); err != nil {
		return err // ErrNotFound when the app is not the tenant's
	}
	return nil
}

// checkMoveTarget validates a move destination: it must belong to the app/tenant
// and must not be the collection itself or one of its descendants (a cycle).
func (s *Collections) checkMoveTarget(ctx context.Context, tx *sqlx.Tx, appID, tenantID, id string, newParentID *string) error {
	if newParentID == nil {
		return nil // the root is always a valid destination
	}
	if *newParentID == id {
		return ErrCollectionCycle
	}
	// Walk the destination's ancestry; encountering id means the destination is
	// inside the subtree being moved.
	cur := newParentID
	for cur != nil {
		c, err := s.Select().
			Where("id", "=", "id").
			Where("app_id", "=", "app_id").
			Where("tenant_id", "=", "tenant_id").
			ExecTx(ctx, tx, map[string]any{"id": *cur, "app_id": appID, "tenant_id": tenantID})
		if err != nil {
			return err // ErrNotFound when the destination is not in the app/tenant
		}
		if c.ID == id {
			return ErrCollectionCycle
		}
		cur = c.ParentID
	}
	return nil
}

// pathOf returns a collection's materialized full path (ancestor names joined by
// "/"), or "" for the app root (id nil). Walks parents within the transaction.
func (s *Collections) pathOf(ctx context.Context, tx *sqlx.Tx, appID, tenantID string, id *string) (string, error) {
	if id == nil {
		return "", nil
	}
	var reversed []string
	cur := id
	for cur != nil {
		c, err := s.Select().
			Where("id", "=", "id").
			Where("app_id", "=", "app_id").
			Where("tenant_id", "=", "tenant_id").
			ExecTx(ctx, tx, map[string]any{"id": *cur, "app_id": appID, "tenant_id": tenantID})
		if err != nil {
			return "", err
		}
		reversed = append(reversed, c.Name)
		cur = c.ParentID
	}
	// reversed is self→root; the path is root→self.
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return strings.Join(reversed, "/"), nil
}

// rewriteDescendantKeys replaces the oldPath prefix with newPath on every
// descendant document key, within tx. Descendants are the documents whose key
// lies under oldPath ("oldPath/..."). A no-op when the path is unchanged.
func (s *Collections) rewriteDescendantKeys(ctx context.Context, tx *sqlx.Tx, appID, tenantID, oldPath, newPath string) error {
	if oldPath == newPath {
		return nil
	}
	pattern := escapeLike(oldPath) + "/%"
	docs, err := s.documents.Query().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Where("key", "LIKE", "pattern").
		ExecTx(ctx, tx, map[string]any{"app_id": appID, "tenant_id": tenantID, "pattern": pattern})
	if err != nil {
		return fmt.Errorf("scanning descendant documents: %w", err)
	}
	now := time.Now()
	for _, doc := range docs {
		newKey := newPath + doc.Key[len(oldPath):] // suffix begins with "/"
		if _, err := s.documents.Modify().
			Set("key", "key").
			Set("updated_at", "updated_at").
			Where("id", "=", "id").
			ExecTx(ctx, tx, map[string]any{"key": newKey, "updated_at": now, "id": doc.ID}); err != nil {
			if isUniqueViolation(err) {
				return ErrCollectionNameTaken // a document already occupies the destination path
			}
			return fmt.Errorf("rewriting document key: %w", err)
		}
	}
	return nil
}

// siblingDocumentExists reports whether a non-deleted document already occupies
// (app, parent, name) — the cross-table half of the sibling namespace.
func (s *Collections) siblingDocumentExists(ctx context.Context, tx *sqlx.Tx, appID, tenantID string, parentID *string, name string) (bool, error) {
	q := s.documents.Count().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Where("name", "=", "name").
		WhereNull("deleted_at")
	args := map[string]any{"app_id": appID, "tenant_id": tenantID, "name": name}
	if parentID != nil {
		q = q.Where("collection_id", "=", "collection_id")
		args["collection_id"] = *parentID
	} else {
		q = q.WhereNull("collection_id")
	}
	n, err := q.ExecTx(ctx, tx, args)
	if err != nil {
		return false, fmt.Errorf("checking sibling documents: %w", err)
	}
	return n > 0, nil
}

// isEmpty reports whether a collection holds no subcollections and no documents.
func (s *Collections) isEmpty(ctx context.Context, tx *sqlx.Tx, appID, tenantID, id string) (bool, error) {
	subs, err := s.Count().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Where("parent_id", "=", "parent_id").
		ExecTx(ctx, tx, map[string]any{"app_id": appID, "tenant_id": tenantID, "parent_id": id})
	if err != nil {
		return false, fmt.Errorf("counting subcollections: %w", err)
	}
	if subs > 0 {
		return false, nil
	}
	docs, err := s.documents.Count().
		Where("tenant_id", "=", "tenant_id").
		Where("collection_id", "=", "collection_id").
		WhereNull("deleted_at").
		ExecTx(ctx, tx, map[string]any{"tenant_id": tenantID, "collection_id": id})
	if err != nil {
		return false, fmt.Errorf("counting documents: %w", err)
	}
	return docs == 0, nil
}

// childCollections lists the direct subcollections of collectionID (nil = root),
// ordered by name. tx may be nil for a non-transactional read.
func (s *Collections) childCollections(ctx context.Context, tx *sqlx.Tx, appID, tenantID string, collectionID *string) ([]*models.Collection, error) {
	q := s.Query().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		OrderBy("name", "asc")
	args := map[string]any{"app_id": appID, "tenant_id": tenantID}
	if collectionID != nil {
		q = q.Where("parent_id", "=", "parent_id")
		args["parent_id"] = *collectionID
	} else {
		q = q.WhereNull("parent_id")
	}
	if tx != nil {
		return q.ExecTx(ctx, tx, args)
	}
	return q.Exec(ctx, args)
}

// childDocuments lists the direct documents of collectionID (nil = root),
// excluding soft-deleted rows, ordered by key. tx may be nil.
func (s *Collections) childDocuments(ctx context.Context, tx *sqlx.Tx, appID, tenantID string, collectionID *string) ([]*models.Document, error) {
	q := s.documents.Query().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		WhereNull("deleted_at").
		OrderBy("key", "asc")
	args := map[string]any{"app_id": appID, "tenant_id": tenantID}
	if collectionID != nil {
		q = q.Where("collection_id", "=", "collection_id")
		args["collection_id"] = *collectionID
	} else {
		q = q.WhereNull("collection_id")
	}
	if tx != nil {
		return q.ExecTx(ctx, tx, args)
	}
	return q.Exec(ctx, args)
}

// inTx runs fn in a transaction, committing on success and rolling back on error.
func (s *Collections) inTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
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

// joinPath appends a segment to a parent path, yielding just the segment at the
// root (empty parent).
func joinPath(parent, segment string) string {
	if parent == "" {
		return segment
	}
	return parent + "/" + segment
}

// escapeLike escapes the LIKE metacharacters in a literal so it matches only
// itself, using the Postgres default escape character (backslash).
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// samePtr reports whether two optional ids are equal (both nil, or both the same
// value).
func samePtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
