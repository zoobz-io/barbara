//go:build testing

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/internal/boot"
)

// searchReadsFixture builds the search store over a freshly-mapped documents
// index and indexes three published projections under testTenant.
func searchReadsFixture(t *testing.T) *stores.Search {
	t.Helper()
	addr := osAddr(t)
	provider := osProvider(t)
	ctx := context.Background()
	sum.Reset() // fresh catalog — NewSearch re-registers "srch://documents"
	sum.New()

	// Recreate the index with the explicit mapping (keyword key/tags, analyzed
	// content) so term and full-text queries behave.
	deleteIndex(t, addr, "documents")
	if err := boot.EnsureIndices(ctx, addr); err != nil {
		t.Fatalf("ensure indices: %v", err)
	}
	t.Cleanup(func() { deleteIndex(t, addr, "documents") })

	store := stores.NewSearch(provider)
	tctx := tenantCtx(testTenant)
	docs := []string{
		`{"document_id":"d1","tenant_id":"` + testTenant + `","key":"guides/install.md","content":"how to install the thing","tags":["guide","setup"],"version_number":1}`,
		`{"document_id":"d2","tenant_id":"` + testTenant + `","key":"guides/config.md","content":"configure the thing carefully","tags":["guide"],"version_number":1}`,
		`{"document_id":"d3","tenant_id":"` + otherTenant + `","key":"guides/install.md","content":"other tenant install","tags":["guide"],"version_number":1}`,
	}
	ids := []string{"d1", "d2", "d3"}
	for i, d := range docs {
		if err := store.Index(tctx, ids[i], []byte(d)); err != nil {
			t.Fatalf("index %s: %v", ids[i], err)
		}
	}
	if err := provider.Refresh(ctx, "documents"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	return store
}

func TestSearch_GetPublishedByKey(t *testing.T) {
	store := searchReadsFixture(t)

	doc, err := store.GetPublishedByKey(tenantCtx(testTenant), "guides/install.md")
	if err != nil {
		t.Fatalf("get by key: %v", err)
	}
	if doc.DocumentID != "d1" {
		t.Errorf("got %s, want d1 (tenant-scoped)", doc.DocumentID)
	}

	// A key that exists only for another tenant is not found.
	if _, err := store.GetPublishedByKey(tenantCtx(testTenant), "nope.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("missing key = %v, want ErrNotFound", err)
	}
	// No tenant is refused.
	if _, err := store.GetPublishedByKey(context.Background(), "guides/install.md"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("no-tenant get = %v, want ErrNoTenant", err)
	}
}

func TestSearch_Enumerate(t *testing.T) {
	store := searchReadsFixture(t)
	ctx := tenantCtx(testTenant)

	all, total, err := store.Enumerate(ctx, "", 50, 0)
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	if total != 2 || len(all) != 2 {
		t.Errorf("enumerate returned %d (total %d), want 2 (the tenant's docs only)", len(all), total)
	}

	tagged, total, err := store.Enumerate(ctx, "setup", 50, 0)
	if err != nil {
		t.Fatalf("enumerate by tag: %v", err)
	}
	if total != 1 || tagged[0].DocumentID != "d1" {
		t.Errorf("tag filter returned %d, want just d1", total)
	}
}

func TestSearch_FullText(t *testing.T) {
	store := searchReadsFixture(t)
	ctx := tenantCtx(testTenant)

	hits, total, err := store.Search(ctx, "install", 50, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || hits[0].DocumentID != "d1" {
		t.Errorf("search 'install' returned %d, want just d1 (tenant-scoped)", total)
	}

	// "configure" matches d2's content.
	hits, _, err = store.Search(ctx, "configure", 50, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].DocumentID != "d2" {
		t.Errorf("search 'configure' did not match d2: %+v", hits)
	}
}

func TestSearch_SearchAll_CrossTenant(t *testing.T) {
	store := searchReadsFixture(t)

	// SearchAll is admin machinery: it is not tenant-scoped, so "install" matches
	// both testTenant's d1 and otherTenant's d3.
	hits, total, err := store.SearchAll(context.Background(), "install", 50, 0)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if total != 2 {
		t.Errorf("SearchAll 'install' returned %d, want 2 (both tenants)", total)
	}
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h.DocumentID] = true
	}
	if !seen["d1"] || !seen["d3"] {
		t.Errorf("SearchAll did not cross tenants: %+v", hits)
	}
}
