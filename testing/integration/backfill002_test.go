//go:build testing

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// The 002 backfill seeds one app per tenant, builds the collection tree from
// key prefixes, places every document, and cuts release 1 from the published
// pointers — idempotently: a second run adds nothing.
func TestBackfill002_SeedsTreeAndRelease1(t *testing.T) {
	db := pgDB(t)
	defer func() { _ = db.Close() }()
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())

	tenant := uuid.NewString()
	ctx := tenantCtx(tenant)

	// A nested key (collection "guides"), published; a root key, draft.
	nested, err := st.Documents.Create(ctx, "guides/install.md")
	if err != nil {
		t.Fatalf("creating nested doc: %v", err)
	}
	v, err := st.Versions.Save(ctx, nested.ID, "# install", 0)
	if err != nil {
		t.Fatalf("saving version: %v", err)
	}
	if _, err := st.Publish(ctx, nested.ID, v.ID); err != nil {
		t.Fatalf("publishing: %v", err)
	}
	root, err := st.Documents.Create(ctx, "readme.md")
	if err != nil {
		t.Fatalf("creating root doc: %v", err)
	}

	bg := context.Background()
	first, err := st.Backfill002(bg)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if first.Tenants < 1 {
		t.Fatalf("first run seeded %d tenants, want >= 1", first.Tenants)
	}

	// One app, named default, pointing at release 1.
	var app struct {
		ID               string  `db:"id"`
		Name             string  `db:"name"`
		CurrentReleaseID *string `db:"current_release_id"`
	}
	if err := db.Get(&app, `SELECT id, name, current_release_id FROM apps WHERE tenant_id = $1`, tenant); err != nil {
		t.Fatalf("reading app: %v", err)
	}
	if app.Name != "default" {
		t.Errorf("app name = %q, want default", app.Name)
	}
	if app.CurrentReleaseID == nil {
		t.Fatal("app has no current release; release 1 was not cut")
	}

	// The collection tree: exactly "guides" for this tenant.
	var collections []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	if err := db.Select(&collections, `SELECT id, name FROM collections WHERE tenant_id = $1`, tenant); err != nil {
		t.Fatalf("reading collections: %v", err)
	}
	if len(collections) != 1 || collections[0].Name != "guides" {
		t.Fatalf("collections = %+v, want exactly [guides]", collections)
	}

	// Placement: nested doc in "guides" named install.md; root doc at the app
	// root (NULL collection) named readme.md; keys unchanged.
	var placed struct {
		CollectionID *string `db:"collection_id"`
		Name         *string `db:"name"`
		Key          string  `db:"key"`
	}
	if err := db.Get(&placed, `SELECT collection_id, name, key FROM documents WHERE id = $1`, nested.ID); err != nil {
		t.Fatalf("reading nested doc: %v", err)
	}
	if placed.CollectionID == nil || *placed.CollectionID != collections[0].ID {
		t.Error("nested doc not placed in the guides collection")
	}
	if placed.Name == nil || *placed.Name != "install.md" {
		t.Errorf("nested doc name = %v, want install.md", placed.Name)
	}
	if placed.Key != "guides/install.md" {
		t.Errorf("nested doc key changed to %q", placed.Key)
	}
	if err := db.Get(&placed, `SELECT collection_id, name, key FROM documents WHERE id = $1`, root.ID); err != nil {
		t.Fatalf("reading root doc: %v", err)
	}
	if placed.CollectionID != nil {
		t.Error("root doc should sit at the app root (NULL collection)")
	}
	if placed.Name == nil || *placed.Name != "readme.md" {
		t.Errorf("root doc name = %v, want readme.md", placed.Name)
	}

	// Release 1 snapshots exactly the published pointer.
	var entries []struct {
		Key       string `db:"key"`
		VersionID string `db:"version_id"`
	}
	if err := db.Select(&entries, `SELECT key, version_id FROM release_entries WHERE release_id = $1`, *app.CurrentReleaseID); err != nil {
		t.Fatalf("reading release entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "guides/install.md" || entries[0].VersionID != v.ID {
		t.Fatalf("release 1 entries = %+v, want exactly the published doc at %s", entries, v.ID)
	}

	// Idempotency: the second run skips the seeded tenant and adds no rows.
	second, err := st.Backfill002(bg)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if second.Tenants != 0 || second.Collections != 0 || second.Releases != 0 {
		t.Errorf("second run should be a no-op for seeded tenants, got %+v", second)
	}
	var appCount int
	if err := db.Get(&appCount, `SELECT count(*) FROM apps WHERE tenant_id = $1`, tenant); err != nil {
		t.Fatalf("counting apps: %v", err)
	}
	if appCount != 1 {
		t.Errorf("apps for tenant = %d after rerun, want 1", appCount)
	}
}
