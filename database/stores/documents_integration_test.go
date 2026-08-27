//go:build testing

package stores

import (
	"context"
	"errors"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/internal/auth"
)

const (
	testTenant  = "11111111-0000-0000-0000-000000000001"
	otherTenant = "22222222-0000-0000-0000-000000000002"
)

// tenantCtx returns a context scoped to the given tenant, the way a handler
// bridges req.Identity before calling a store.
func tenantCtx(tenantID string) context.Context {
	p := auth.NewPrincipal("user-1", tenantID, "", nil, nil)
	return auth.WithPrincipal(context.Background(), p)
}

func TestDocuments_CreateGetList(t *testing.T) {
	db := jobsDB(t) // real Postgres + fresh sum catalog
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM documents") })
	store := NewDocuments(db, astqlpg.New())
	ctx := tenantCtx(testTenant)

	doc, err := store.Create(ctx, "guides/install.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if doc.ID == "" {
		t.Error("create did not return a generated ID")
	}
	if doc.Key != "guides/install.md" || doc.TenantID != testTenant {
		t.Errorf("unexpected doc: %+v", doc)
	}
	if doc.CreatedAt.IsZero() || doc.UpdatedAt.IsZero() {
		t.Errorf("timestamps not populated: %+v", doc)
	}

	// Get is tenant-scoped.
	got, err := store.Get(ctx, doc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != doc.ID {
		t.Errorf("get returned %s, want %s", got.ID, doc.ID)
	}
	if _, err := store.Get(tenantCtx(otherTenant), doc.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-tenant get = %v, want ErrNotFound", err)
	}

	// A store call with no tenant is refused.
	if _, err := store.Get(context.Background(), doc.ID); !errors.Is(err, ErrNoTenant) {
		t.Errorf("get without tenant = %v, want ErrNoTenant", err)
	}

	// Duplicate key for the same tenant is rejected.
	if _, err := store.Create(ctx, "guides/install.md"); err == nil {
		t.Error("expected duplicate key to be rejected")
	}

	list, err := store.List(ctx, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("list returned %d, want 1", len(list))
	}
}

func TestDocuments_RenameFreesKey(t *testing.T) {
	db := jobsDB(t)
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM documents") })
	store := NewDocuments(db, astqlpg.New())
	ctx := tenantCtx(testTenant)

	doc, err := store.Create(ctx, "old-key.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	renamed, err := store.Rename(ctx, doc.ID, "new-key.md")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Key != "new-key.md" {
		t.Errorf("key = %q, want new-key.md", renamed.Key)
	}

	// The old key is free — a new document can claim it.
	if _, err := store.Create(ctx, "old-key.md"); err != nil {
		t.Errorf("old key should be free after rename: %v", err)
	}
}

func TestDocuments_DeleteGuardsPublished(t *testing.T) {
	db := jobsDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM versions")
		_, _ = db.Exec("DELETE FROM documents")
	})
	store := NewDocuments(db, astqlpg.New())
	ctx := tenantCtx(testTenant)

	doc, err := store.Create(ctx, "publishable.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Unpublished delete succeeds and the document is gone.
	dup, err := store.Create(ctx, "deletable.md")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.Delete(ctx, dup.ID); err != nil {
		t.Fatalf("delete unpublished: %v", err)
	}
	if _, err := store.Get(ctx, dup.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted doc still gettable: %v", err)
	}

	// Publish doc by pointing at a real version, then deletion must be refused.
	var versionID string
	if err := db.QueryRowx(
		`INSERT INTO versions (document_id, tenant_id, version_number, content, created_by)
		 VALUES ($1, $2, 1, 'hi', gen_random_uuid()) RETURNING id`,
		doc.ID, testTenant).Scan(&versionID); err != nil {
		t.Fatalf("insert version: %v", err)
	}
	if _, err := db.Exec("UPDATE documents SET published_version_id = $1 WHERE id = $2", versionID, doc.ID); err != nil {
		t.Fatalf("set published pointer: %v", err)
	}

	if err := store.Delete(ctx, doc.ID); !errors.Is(err, ErrDocumentPublished) {
		t.Errorf("delete published = %v, want ErrDocumentPublished", err)
	}

	// A missing document reports not found.
	if err := store.Delete(ctx, "33333333-0000-0000-0000-000000000003"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing = %v, want ErrNotFound", err)
	}
}
