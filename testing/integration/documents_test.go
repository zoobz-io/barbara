//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

const (
	testTenant  = "11111111-0000-0000-0000-000000000001"
	otherTenant = "22222222-0000-0000-0000-000000000002"
)

// newDocStores builds a fully-wired stores aggregate over a real database for
// the document tree tests, returning the shared connection for raw setup and a
// cleanup that clears every table in FK order.
func newDocStores(t *testing.T) (*stores.Stores, *sqlx.DB, func()) {
	t.Helper()
	db := pgDB(t)
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	cleanup := func() {
		// FK order: clear the app→release pointer, drop release_entries (RESTRICT
		// refs to documents/versions), then releases; deleting documents cascades
		// their versions — deleting versions first would trip the published-pointer
		// RESTRICT.
		_, _ = db.Exec("UPDATE apps SET current_release_id = NULL")
		_, _ = db.Exec("DELETE FROM release_entries")
		_, _ = db.Exec("DELETE FROM releases")
		_, _ = db.Exec("DELETE FROM documents")
		_, _ = db.Exec("DELETE FROM collections")
		_, _ = db.Exec("DELETE FROM apps")
		_ = db.Close()
	}
	return st, db, cleanup
}

// insertVersion adds a version for a document and returns its id.
func insertVersion(t *testing.T, db *sqlx.DB, docID string) string {
	t.Helper()
	var id string
	if err := db.QueryRowx(
		`INSERT INTO versions (document_id, tenant_id, version_number, content, created_by)
		 VALUES ($1, $2, 1, 'hi', gen_random_uuid()) RETURNING id`,
		docID, testTenant).Scan(&id); err != nil {
		t.Fatalf("insert version: %v", err)
	}
	return id
}

func insertRelease(t *testing.T, db *sqlx.DB, appID string, number int) string {
	t.Helper()
	var id string
	if err := db.QueryRowx(
		`INSERT INTO releases (app_id, tenant_id, number, created_by)
		 VALUES ($1, $2, $3, gen_random_uuid()) RETURNING id`,
		appID, testTenant, number).Scan(&id); err != nil {
		t.Fatalf("insert release: %v", err)
	}
	return id
}

func insertReleaseEntry(t *testing.T, db *sqlx.DB, releaseID, key, docID, versionID string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO release_entries (release_id, key, document_id, version_id)
		 VALUES ($1, $2, $3, $4)`,
		releaseID, key, docID, versionID); err != nil {
		t.Fatalf("insert release entry: %v", err)
	}
}

func countVersions(t *testing.T, db *sqlx.DB, docID string) int {
	t.Helper()
	var n int
	if err := db.QueryRowx(`SELECT count(*) FROM versions WHERE document_id=$1`, docID).Scan(&n); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	return n
}

func hasDeletedAt(t *testing.T, db *sqlx.DB, docID string) bool {
	t.Helper()
	var deleted bool
	if err := db.QueryRowx(`SELECT deleted_at IS NOT NULL FROM documents WHERE id=$1`, docID).Scan(&deleted); err != nil {
		t.Fatalf("check deleted_at: %v", err)
	}
	return deleted
}

// Create places a document in the tree: the key is materialized from the
// collection path and the name. Sibling names are unique; cross-tenant reads
// scope out.
func TestDocuments_TreePlacement(t *testing.T) {
	st, _, cleanup := newDocStores(t)
	t.Cleanup(cleanup)
	ctx := tenantCtx(testTenant)

	app, err := st.Apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	guides, err := st.Collections.Create(ctx, app.ID, nil, "guides")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}

	// A document under guides gets the materialized key guides/install.md.
	doc, err := st.Documents.Create(ctx, app.ID, &guides.ID, "install.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if doc.Key != "guides/install.md" || doc.CollectionID == nil || *doc.CollectionID != guides.ID {
		t.Errorf("unexpected placement: key=%q collection=%v", doc.Key, doc.CollectionID)
	}
	if doc.CreatedAt.IsZero() {
		t.Errorf("timestamps not populated: %+v", doc)
	}

	// A root document keys to just its name.
	root, err := st.Documents.Create(ctx, app.ID, nil, "readme.md")
	if err != nil {
		t.Fatalf("create root doc: %v", err)
	}
	if root.Key != "readme.md" {
		t.Errorf("root key = %q, want readme.md", root.Key)
	}

	// Cross-tenant get scopes out; a no-tenant call is refused.
	if _, err := st.Documents.Get(tenantCtx(otherTenant), doc.ID); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("cross-tenant get = %v, want ErrNotFound", err)
	}
	if _, err := st.Documents.Get(context.Background(), doc.ID); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("get without tenant = %v, want ErrNoTenant", err)
	}

	// A sibling document with the same name collides (same key).
	if _, err := st.Documents.Create(ctx, app.ID, &guides.ID, "install.md"); !errors.Is(err, stores.ErrCollectionNameTaken) {
		t.Errorf("duplicate sibling = %v, want ErrCollectionNameTaken", err)
	}
	// So does a collection sharing the name.
	if _, err := st.Documents.Create(ctx, app.ID, nil, "guides"); !errors.Is(err, stores.ErrCollectionNameTaken) {
		t.Errorf("document colliding with sibling collection = %v, want ErrCollectionNameTaken", err)
	}
}

// Move reparents and renames a document, rewriting the materialized key each time.
func TestDocuments_Move(t *testing.T) {
	st, _, cleanup := newDocStores(t)
	t.Cleanup(cleanup)
	ctx := tenantCtx(testTenant)

	app, _ := st.Apps.Create(ctx, "site")
	guides, _ := st.Collections.Create(ctx, app.ID, nil, "guides")
	manuals, _ := st.Collections.Create(ctx, app.ID, nil, "manuals")
	doc, err := st.Documents.Create(ctx, app.ID, &guides.ID, "install.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Move to another collection: key follows.
	moved, err := st.Documents.Move(ctx, app.ID, doc.ID, &manuals.ID, "install.md")
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.Key != "manuals/install.md" {
		t.Errorf("after move, key = %q, want manuals/install.md", moved.Key)
	}

	// Rename in place (same collection, new name).
	renamed, err := st.Documents.Move(ctx, app.ID, doc.ID, &manuals.ID, "setup.md")
	if err != nil {
		t.Fatalf("rename via move: %v", err)
	}
	if renamed.Key != "manuals/setup.md" {
		t.Errorf("after rename, key = %q, want manuals/setup.md", renamed.Key)
	}

	// Move to the app root.
	toRoot, err := st.Documents.Move(ctx, app.ID, doc.ID, nil, "setup.md")
	if err != nil {
		t.Fatalf("move to root: %v", err)
	}
	if toRoot.Key != "setup.md" {
		t.Errorf("after move to root, key = %q, want setup.md", toRoot.Key)
	}
}

// Delete: hard-delete an unreferenced document (versions cascade), refuse a
// published one, and soft-delete one a historical release references (freeing
// the key, keeping the versions).
func TestDocuments_DeleteRules(t *testing.T) {
	st, db, cleanup := newDocStores(t)
	t.Cleanup(cleanup)
	ctx := tenantCtx(testTenant)

	app, _ := st.Apps.Create(ctx, "site")

	// Hard delete: an unreferenced document with a version — the version cascades.
	hard, _ := st.Documents.Create(ctx, app.ID, nil, "scratch.md")
	insertVersion(t, db, hard.ID)
	if err := st.Documents.Delete(ctx, hard.ID); err != nil {
		t.Fatalf("hard delete: %v", err)
	}
	if _, err := st.Documents.Get(ctx, hard.ID); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("hard-deleted doc still present: %v", err)
	}
	if n := countVersions(t, db, hard.ID); n != 0 {
		t.Errorf("versions not cascaded: %d remain", n)
	}

	// Refuse: a published document.
	pub, _ := st.Documents.Create(ctx, app.ID, nil, "live.md")
	vid := insertVersion(t, db, pub.ID)
	if _, err := db.Exec("UPDATE documents SET published_version_id=$1 WHERE id=$2", vid, pub.ID); err != nil {
		t.Fatalf("set pointer: %v", err)
	}
	if err := st.Documents.Delete(ctx, pub.ID); !errors.Is(err, stores.ErrDocumentPublished) {
		t.Errorf("delete published = %v, want ErrDocumentPublished", err)
	}

	// Soft delete: a document referenced only by a historical release.
	soft, _ := st.Documents.Create(ctx, app.ID, nil, "archived.md")
	svid := insertVersion(t, db, soft.ID)
	r1 := insertRelease(t, db, app.ID, 1)
	insertReleaseEntry(t, db, r1, "archived.md", soft.ID, svid)
	r2 := insertRelease(t, db, app.ID, 2) // r2 does not carry the doc, and is current
	if _, err := db.Exec("UPDATE apps SET current_release_id=$1 WHERE id=$2", r2, app.ID); err != nil {
		t.Fatalf("point app at r2: %v", err)
	}

	if err := st.Documents.Delete(ctx, soft.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	// The row survives with deleted_at set; its version survives.
	if !hasDeletedAt(t, db, soft.ID) {
		t.Error("soft-deleted doc has no deleted_at")
	}
	if n := countVersions(t, db, soft.ID); n != 1 {
		t.Errorf("soft delete dropped versions: %d remain, want 1", n)
	}
	// The key is freed: a new document can take it.
	if _, err := st.Documents.Create(ctx, app.ID, nil, "archived.md"); err != nil {
		t.Errorf("key not freed after soft delete: %v", err)
	}

	// A document carried by the CURRENT release is refused.
	live, _ := st.Documents.Create(ctx, app.ID, nil, "current.md")
	lvid := insertVersion(t, db, live.ID)
	insertReleaseEntry(t, db, r2, "current.md", live.ID, lvid)
	if err := st.Documents.Delete(ctx, live.ID); !errors.Is(err, stores.ErrDocumentPublished) {
		t.Errorf("delete of a current-release doc = %v, want ErrDocumentPublished", err)
	}
}
