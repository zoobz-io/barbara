//go:build testing

package integration

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
)

// placeDoc inserts a document already placed in the tree — app, collection
// (empty = root), name, and materialized key — the way #61 will place documents
// once that ticket lands. Returns the generated id.
func placeDoc(t *testing.T, db *sqlx.DB, appID, collectionID, name, key string) string {
	t.Helper()
	var collID any
	if collectionID != "" {
		collID = collectionID
	}
	var id string
	if err := db.QueryRowx(
		`INSERT INTO documents (tenant_id, key, tags, app_id, collection_id, name)
		 VALUES ($1, $2, '{}', $3, $4, $5) RETURNING id`,
		testTenant, key, appID, collID, name).Scan(&id); err != nil {
		t.Fatalf("place doc %q: %v", key, err)
	}
	return id
}

// docKey re-reads a document's current key.
func docKey(t *testing.T, documents *stores.Documents, id string) string {
	t.Helper()
	d, err := documents.Get(tenantCtx(testTenant), id)
	if err != nil {
		t.Fatalf("get doc %s: %v", id, err)
	}
	return d.Key
}

// TestCollections_TreeRewriteAndGuards exercises the plan-002 tree mechanics
// against real Postgres: building a tree, the descendant-key rewrite on rename
// and move, the cross-table sibling namespace, and the delete guards. This is
// the logic query-generation unit tests cannot prove — that the prefix
// arithmetic and the LIKE scan actually produce the right keys.
func TestCollections_TreeRewriteAndGuards(t *testing.T) {
	db := pgDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM documents")
		_, _ = db.Exec("DELETE FROM collections")
		_, _ = db.Exec("DELETE FROM apps")
		_ = db.Close()
	})

	renderer := astqlpg.New()
	documents := stores.NewDocuments(db, renderer)
	apps := stores.NewApps(db, renderer)
	cols := stores.NewCollections(db, renderer, documents, apps)
	ctx := tenantCtx(testTenant)

	app, err := apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Build guides/api and place a document at guides/api/ref.md.
	guides, err := cols.Create(ctx, app.ID, nil, "guides")
	if err != nil {
		t.Fatalf("create guides: %v", err)
	}
	api, err := cols.Create(ctx, app.ID, &guides.ID, "api")
	if err != nil {
		t.Fatalf("create api: %v", err)
	}
	doc := placeDoc(t, db, app.ID, api.ID, "ref.md", "guides/api/ref.md")

	// Rename the top collection: every descendant key's prefix rewrites.
	if _, err := cols.Rename(ctx, app.ID, guides.ID, "manuals"); err != nil {
		t.Fatalf("rename guides->manuals: %v", err)
	}
	if got := docKey(t, documents, doc); got != "manuals/api/ref.md" {
		t.Errorf("after rename, key = %q, want manuals/api/ref.md", got)
	}

	// Move api to the app root: the prefix collapses to just the collection.
	if _, err := cols.Move(ctx, app.ID, api.ID, nil); err != nil {
		t.Fatalf("move api to root: %v", err)
	}
	if got := docKey(t, documents, doc); got != "api/ref.md" {
		t.Errorf("after move, key = %q, want api/ref.md", got)
	}

	// A sibling collection may not share a name with a sibling document. Place a
	// document "notes" at the root, then a collection "notes" at the root fails.
	_ = placeDoc(t, db, app.ID, "", "notes", "notes")
	if _, err := cols.Create(ctx, app.ID, nil, "notes"); !errors.Is(err, stores.ErrCollectionNameTaken) {
		t.Errorf("collection colliding with a sibling document = %v, want ErrCollectionNameTaken", err)
	}

	// api still holds ref.md, so it will not delete...
	if err := cols.Delete(ctx, app.ID, api.ID); !errors.Is(err, stores.ErrCollectionNotEmpty) {
		t.Errorf("delete non-empty = %v, want ErrCollectionNotEmpty", err)
	}
	// ...but the now-empty guides (manuals) does.
	if err := cols.Delete(ctx, app.ID, guides.ID); err != nil {
		t.Errorf("delete empty collection: %v", err)
	}

	// Cycle guard: guides is gone; make a fresh chain a→b and try to move a under b.
	a, err := cols.Create(ctx, app.ID, nil, "a")
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := cols.Create(ctx, app.ID, &a.ID, "b")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	if _, err := cols.Move(ctx, app.ID, a.ID, &b.ID); !errors.Is(err, stores.ErrCollectionCycle) {
		t.Errorf("move a under its descendant b = %v, want ErrCollectionCycle", err)
	}
}

// TestCollections_ListContents lists the root and a collection in one call each.
func TestCollections_ListContents(t *testing.T) {
	db := pgDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM documents")
		_, _ = db.Exec("DELETE FROM collections")
		_, _ = db.Exec("DELETE FROM apps")
		_ = db.Close()
	})

	renderer := astqlpg.New()
	documents := stores.NewDocuments(db, renderer)
	apps := stores.NewApps(db, renderer)
	cols := stores.NewCollections(db, renderer, documents, apps)
	ctx := tenantCtx(testTenant)

	app, err := apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	guides, err := cols.Create(ctx, app.ID, nil, "guides")
	if err != nil {
		t.Fatalf("create guides: %v", err)
	}
	_ = placeDoc(t, db, app.ID, "", "readme.md", "readme.md")     // root document
	_ = placeDoc(t, db, app.ID, guides.ID, "intro.md", "guides/intro.md")

	root, err := cols.ListContents(ctx, app.ID, nil)
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	if len(root.Subcollections) != 1 || root.Subcollections[0].Name != "guides" {
		t.Errorf("root subcollections = %+v, want [guides]", root.Subcollections)
	}
	if len(root.Documents) != 1 || root.Documents[0].Key != "readme.md" {
		t.Errorf("root documents = %+v, want [readme.md]", root.Documents)
	}

	inside, err := cols.ListContents(ctx, app.ID, &guides.ID)
	if err != nil {
		t.Fatalf("list guides: %v", err)
	}
	if len(inside.Subcollections) != 0 || len(inside.Documents) != 1 || inside.Documents[0].Key != "guides/intro.md" {
		t.Errorf("guides contents = %+v, want just guides/intro.md", inside)
	}
}
