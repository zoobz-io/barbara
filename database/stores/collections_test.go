//go:build testing

package stores

import (
	"context"
	"errors"
	"testing"

	"github.com/lib/pq"

	"github.com/zoobz-io/grub/mockdb"

	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

const testApp = "app-1"

// collectionRow is a collections row. parentID may be nil (root) or a string.
func collectionRow(id, name string, parentID any) *mockdb.RowData {
	return &mockdb.RowData{
		Columns: []string{"id", "tenant_id", "app_id", "parent_id", "name"},
		Rows:    [][]any{{id, testTenant, testApp, parentID, name}},
	}
}

// Create at the app root: validate the app is the tenant's, confirm no sibling
// document holds the name, then insert the collection.
func TestCollections_Create_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())          // requireParentScope: apps.Get
	cfg.PushRowData(countRow(0))       // siblingDocumentExists: no doc sibling
	cfg.PushRowData(collectionRow("c-1", "guides", nil)) // INSERT ... RETURNING

	_, err := st.Collections.Create(tenantCtx(), testApp, nil, "guides")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// The sibling check counts documents at (app, root, name), excluding deleted.
	sib := queryAt(t, capture, 1)
	wantSQL(t, sib, `SELECT COUNT(*) FROM "documents"`, `"app_id" = ?`, `"tenant_id" = ?`,
		`"name" = ?`, `"collection_id" IS NULL`, `"deleted_at" IS NULL`)
	wantArg(t, sib, "guides")

	ins := queryAt(t, capture, 2)
	wantSQL(t, ins, `INSERT INTO "collections"`, `"app_id"`, `"tenant_id"`, `"name"`, `RETURNING`)
	wantArg(t, ins, testApp)
	wantArg(t, ins, "guides")
}

// A sibling document with the same name blocks the create with ErrCollectionNameTaken,
// before any INSERT.
func TestCollections_Create_SiblingDocumentBlocks(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())    // scope ok
	cfg.PushRowData(countRow(1)) // a document sibling already holds the name

	_, err := st.Collections.Create(tenantCtx(), testApp, nil, "guides")
	if !errors.Is(err, ErrCollectionNameTaken) {
		t.Fatalf("create with a sibling doc = %v, want ErrCollectionNameTaken", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "collections"`)
	}
}

// Get is scoped to id, app, and tenant.
func TestCollections_Get_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Collections.Get(tenantCtx(), testApp, "c-1")

	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "collections"`, `"id" = ?`, `"app_id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "c-1")
	wantArg(t, q, testApp)
}

// ListContents at the root: after validating the app, it lists root collections
// (parent_id IS NULL) and root documents (collection_id IS NULL, not deleted).
func TestCollections_ListContents_Root_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow()) // requireParentScope: apps.Get

	_, err := st.Collections.ListContents(tenantCtx(), testApp, nil)
	if err != nil {
		t.Fatalf("list contents: %v", err)
	}

	cols := queryAt(t, capture, 1)
	wantSQL(t, cols, `FROM "collections"`, `"app_id" = ?`, `"parent_id" IS NULL`, `ORDER BY "name" ASC`)

	docs := queryAt(t, capture, 2)
	wantSQL(t, docs, `FROM "documents"`, `"app_id" = ?`, `"collection_id" IS NULL`,
		`"deleted_at" IS NULL`, `ORDER BY "key" ASC`)
}

// Rename a root collection: lock, sibling check, update the name, then rewrite
// descendant document keys under the old path prefix.
func TestCollections_Rename_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil))     // lock (FOR UPDATE)
	cfg.PushRowData(countRow(0))                             // sibling check
	cfg.PushRowData(collectionRow("c-1", "tutorials", nil))  // UPDATE ... RETURNING
	// rewrite descendant SELECT returns no rows (no pushed data)

	_, err := st.Collections.Rename(tenantCtx(), testApp, "c-1", "tutorials")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}

	lock := queryAt(t, capture, 0)
	wantSQL(t, lock, `FROM "collections"`, `"id" = ?`, `"app_id" = ?`, `"tenant_id" = ?`, `FOR UPDATE`)

	upd := queryAt(t, capture, 2)
	wantSQL(t, upd, `UPDATE "collections" SET`, `"name" = ?`, `"id" = ?`, `"app_id" = ?`, `RETURNING`)
	wantArg(t, upd, "tutorials")

	rewrite := queryAt(t, capture, 3)
	wantSQL(t, rewrite, `FROM "documents"`, `"key" LIKE ?`, `"app_id" = ?`)
	wantArg(t, rewrite, "guides/%")
}

// A no-op rename (same name) touches nothing past the lock and emits no event.
func TestCollections_Rename_NoOp(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil)) // lock

	fired := false
	l := events.Collection.Renamed.Listen(func(_ context.Context, _ events.CollectionRenamedEvent) { fired = true })
	defer l.Close()

	if _, err := st.Collections.Rename(tenantCtx(), testApp, "c-1", "guides"); err != nil {
		t.Fatalf("rename no-op: %v", err)
	}
	if fired {
		t.Error("Collection.Renamed emitted for a same-name rename")
	}
	if len(capture.Queries) != 1 {
		t.Errorf("a no-op rename issued %d queries, want 1 (the lock)", len(capture.Queries))
	}
}

// Move a root collection under a new parent: lock, walk the destination ancestry
// (cycle guard), sibling check, resolve the new parent path, update parent_id,
// then rewrite descendant keys.
func TestCollections_Move_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil))   // lock
	cfg.PushRowData(collectionRow("p-1", "manuals", nil))  // checkMoveTarget: walk p-1 (parent nil → stop)
	cfg.PushRowData(countRow(0))                           // sibling check at the new parent
	cfg.PushRowData(collectionRow("p-1", "manuals", nil))  // pathOf(new parent)
	cfg.PushRowData(collectionRow("c-1", "guides", "p-1")) // UPDATE ... RETURNING
	// rewrite descendant SELECT returns no rows

	newParent := "p-1"
	_, err := st.Collections.Move(tenantCtx(), testApp, "c-1", &newParent)
	if err != nil {
		t.Fatalf("move: %v", err)
	}

	upd := queryAt(t, capture, 4)
	wantSQL(t, upd, `UPDATE "collections" SET`, `"parent_id" = ?`, `"id" = ?`, `RETURNING`)
	wantArg(t, upd, "p-1")

	rewrite := queryAt(t, capture, 5)
	wantSQL(t, rewrite, `FROM "documents"`, `"key" LIKE ?`)
	wantArg(t, rewrite, "guides/%")
}

// Moving a collection into itself is a cycle, refused after the lock.
func TestCollections_Move_SelfIsCycle(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil)) // lock

	self := "c-1"
	_, err := st.Collections.Move(tenantCtx(), testApp, "c-1", &self)
	if !errors.Is(err, ErrCollectionCycle) {
		t.Fatalf("move into self = %v, want ErrCollectionCycle", err)
	}
}

// Moving a collection under one of its descendants is a cycle: the destination's
// ancestry walk reaches the collection being moved.
func TestCollections_Move_IntoDescendantIsCycle(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil))    // lock target c-1
	cfg.PushRowData(collectionRow("d-1", "sub", "c-1"))     // walk: d-1's parent is c-1...
	// next walk step fetches c-1, whose id == target → cycle
	cfg.PushRowData(collectionRow("c-1", "guides", nil))

	dest := "d-1"
	_, err := st.Collections.Move(tenantCtx(), testApp, "c-1", &dest)
	if !errors.Is(err, ErrCollectionCycle) {
		t.Fatalf("move into descendant = %v, want ErrCollectionCycle", err)
	}
}

// Delete an empty collection: lock, confirm no subcollections and no documents,
// then remove it.
func TestCollections_Delete_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil)) // lock
	cfg.PushRowData(countRow(0))                         // subcollection count
	cfg.PushRowData(countRow(0))                         // document count

	if err := st.Collections.Delete(tenantCtx(), testApp, "c-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	subs := queryAt(t, capture, 1)
	wantSQL(t, subs, `SELECT COUNT(*) FROM "collections"`, `"parent_id" = ?`)
	docs := queryAt(t, capture, 2)
	wantSQL(t, docs, `SELECT COUNT(*) FROM "documents"`, `"collection_id" = ?`, `"deleted_at" IS NULL`)
	del := queryAt(t, capture, 3)
	wantSQL(t, del, `DELETE FROM "collections"`, `"id" = ?`, `"app_id" = ?`, `"tenant_id" = ?`)
}

// A non-empty collection (a subcollection present) is refused with
// ErrCollectionNotEmpty, before the DELETE.
func TestCollections_Delete_RefusesNonEmpty(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil)) // lock
	cfg.PushRowData(countRow(1))                         // a subcollection exists

	err := st.Collections.Delete(tenantCtx(), testApp, "c-1")
	if !errors.Is(err, ErrCollectionNotEmpty) {
		t.Fatalf("delete non-empty = %v, want ErrCollectionNotEmpty", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `DELETE FROM "collections"`)
	}
}

// Create at the root validates the app is the tenant's: an unknown app scopes
// out with ErrNotFound, before any INSERT.
func TestCollections_Create_UnknownAppScopesOut(t *testing.T) {
	st, capture := newQueryTest(t) // apps.Get finds no row → ErrNotFound
	_, err := st.Collections.Create(tenantCtx(), testApp, nil, "guides")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("create under an unknown app = %v, want ErrNotFound", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "collections"`)
	}
}

// Moving a collection to the parent it already has is a no-op: nothing past the
// lock, no event.
func TestCollections_Move_SameParentNoOp(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", "p-1")) // lock: already under p-1

	fired := false
	l := events.Collection.Moved.Listen(func(_ context.Context, _ events.CollectionMovedEvent) { fired = true })
	defer l.Close()

	parent := "p-1"
	if _, err := st.Collections.Move(tenantCtx(), testApp, "c-1", &parent); err != nil {
		t.Fatalf("move to same parent: %v", err)
	}
	if fired {
		t.Error("Collection.Moved emitted for a same-parent move")
	}
	if len(capture.Queries) != 1 {
		t.Errorf("a same-parent move issued %d queries, want 1 (the lock)", len(capture.Queries))
	}
}

// If rewriting a descendant key collides with an existing document at the
// destination (unique key), the rename fails with ErrCollectionNameTaken.
func TestCollections_Rename_DescendantKeyCollision(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil))    // lock
	cfg.PushRowData(countRow(0))                            // sibling check
	cfg.PushRowData(collectionRow("c-1", "tutorials", nil)) // UPDATE ... RETURNING
	cfg.PushRowData(&mockdb.RowData{ // the one descendant document to rewrite
		Columns: []string{"id", "tenant_id", "key"},
		Rows:    [][]any{{"d-1", testTenant, "guides/sub.md"}},
	})
	cfg.PushQueryErr(&pq.Error{Code: "23505"}) // its key rewrite (UPDATE ... RETURNING) collides

	_, err := st.Collections.Rename(tenantCtx(), testApp, "c-1", "tutorials")
	if !errors.Is(err, ErrCollectionNameTaken) {
		t.Fatalf("rename with a colliding descendant key = %v, want ErrCollectionNameTaken", err)
	}
}

// A child added between the emptiness check and the delete surfaces at the DB as
// a foreign-key violation, translated back to ErrCollectionNotEmpty.
func TestCollections_Delete_ForeignKeyRaceIsNotEmpty(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil)) // lock
	cfg.PushRowData(countRow(0))                         // subcollection count
	cfg.PushRowData(countRow(0))                         // document count
	cfg.PushExecErr(&pq.Error{Code: "23503"})            // ...but the DELETE hits a child FK

	if err := st.Collections.Delete(tenantCtx(), testApp, "c-1"); !errors.Is(err, ErrCollectionNotEmpty) {
		t.Fatalf("delete racing a child insert = %v, want ErrCollectionNotEmpty", err)
	}
}

// Every store method requires a tenant in context.
func TestCollections_RequiresTenant(t *testing.T) {
	st, _ := newQueryTest(t)
	ctx := context.Background()
	if _, err := st.Collections.Create(ctx, testApp, nil, "x"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Create tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Collections.Get(ctx, testApp, "c-1"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Get tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Collections.ListContents(ctx, testApp, nil); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("ListContents tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Collections.Rename(ctx, testApp, "c-1", "y"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Rename tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Collections.Move(ctx, testApp, "c-1", nil); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Move tenantless = %v, want ErrNoTenant", err)
	}
	if err := st.Collections.Delete(ctx, testApp, "c-1"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Delete tenantless = %v, want ErrNoTenant", err)
	}
}

// Create emits Collection.Created on success.
func TestCollections_Create_EmitsCreated(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())
	cfg.PushRowData(countRow(0))
	cfg.PushRowData(collectionRow("c-1", "guides", nil))

	var got events.CollectionCreatedEvent
	fired := false
	l := events.Collection.Created.Listen(func(_ context.Context, e events.CollectionCreatedEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if _, err := st.Collections.Create(tenantCtx(), testApp, nil, "guides"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !fired || got.CollectionID != "c-1" || got.AppID != testApp || got.Name != "guides" {
		t.Errorf("Created event = %+v (fired=%v)", got, fired)
	}
}

// Delete emits Collection.Deleted on success.
func TestCollections_Delete_EmitsDeleted(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(collectionRow("c-1", "guides", nil))
	cfg.PushRowData(countRow(0))
	cfg.PushRowData(countRow(0))

	var got events.CollectionDeletedEvent
	fired := false
	l := events.Collection.Deleted.Listen(func(_ context.Context, e events.CollectionDeletedEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if err := st.Collections.Delete(tenantCtx(), testApp, "c-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !fired || got.CollectionID != "c-1" || got.AppID != testApp {
		t.Errorf("Deleted event = %+v (fired=%v)", got, fired)
	}
}
