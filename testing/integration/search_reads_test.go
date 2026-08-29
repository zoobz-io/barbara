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

// Fixed app ids for the seeded projections. The index has no FK — these only
// need to be stable strings the reads can scope by.
const (
	testApp  = "app-test"
	otherApp = "app-other"
)

// searchReadsFixture builds the search store over a freshly-mapped documents
// index and indexes published projections: three under testTenant (two apps,
// one root doc) and one under otherTenant.
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
		`{"document_id":"d1","tenant_id":"` + testTenant + `","app_id":"` + testApp + `","key":"guides/install.md","parent_path":"guides","content":"how to install the thing","tags":["guide","setup"],"version_number":1}`,
		`{"document_id":"d2","tenant_id":"` + testTenant + `","app_id":"` + testApp + `","key":"guides/config.md","parent_path":"guides","content":"configure the thing carefully","tags":["guide"],"version_number":1}`,
		`{"document_id":"d4","tenant_id":"` + testTenant + `","app_id":"` + testApp + `","key":"readme.md","parent_path":"","content":"the front page","tags":[],"version_number":1}`,
		`{"document_id":"d5","tenant_id":"` + testTenant + `","app_id":"` + otherApp + `","key":"guides/install.md","parent_path":"guides","content":"install the OTHER product","tags":["guide"],"version_number":1}`,
		`{"document_id":"d3","tenant_id":"` + otherTenant + `","app_id":"` + testApp + `","key":"guides/install.md","parent_path":"guides","content":"other tenant install","tags":["guide"],"version_number":1}`,
	}
	ids := []string{"d1", "d2", "d4", "d5", "d3"}
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

	doc, err := store.GetPublishedByKey(tenantCtx(testTenant), testApp, "guides/install.md")
	if err != nil {
		t.Fatalf("get by key: %v", err)
	}
	if doc.DocumentID != "d1" {
		t.Errorf("got %s, want d1 (tenant- and app-scoped)", doc.DocumentID)
	}

	// The same key in another app of the same tenant is a different document.
	doc, err = store.GetPublishedByKey(tenantCtx(testTenant), otherApp, "guides/install.md")
	if err != nil || doc.DocumentID != "d5" {
		t.Errorf("other-app get = %+v,%v; want d5 (app-scoped)", doc, err)
	}

	// A key that does not exist for the app is not found.
	if _, err := store.GetPublishedByKey(tenantCtx(testTenant), testApp, "nope.md"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("missing key = %v, want ErrNotFound", err)
	}
	// No tenant is refused.
	if _, err := store.GetPublishedByKey(context.Background(), testApp, "guides/install.md"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("no-tenant get = %v, want ErrNoTenant", err)
	}
}

func TestSearch_Enumerate(t *testing.T) {
	store := searchReadsFixture(t)
	ctx := tenantCtx(testTenant)

	all, total, err := store.Enumerate(ctx, testApp, "", 50, 0)
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Errorf("enumerate returned %d (total %d), want 3 (the app's docs only)", len(all), total)
	}

	tagged, total, err := store.Enumerate(ctx, testApp, "setup", 50, 0)
	if err != nil {
		t.Fatalf("enumerate by tag: %v", err)
	}
	if total != 1 || tagged[0].DocumentID != "d1" {
		t.Errorf("tag filter returned %d, want just d1", total)
	}
}

func TestSearch_ListFolder(t *testing.T) {
	store := searchReadsFixture(t)
	ctx := tenantCtx(testTenant)

	// A folder listing is its direct children, ordered by key — d5 (other app)
	// and d3 (other tenant) share the parent_path and must not leak in.
	docs, total, err := store.ListFolder(ctx, testApp, "guides", 50, 0)
	if err != nil {
		t.Fatalf("list folder: %v", err)
	}
	if total != 2 || len(docs) != 2 || docs[0].DocumentID != "d2" || docs[1].DocumentID != "d1" {
		t.Errorf("guides folder = %+v (total %d), want [d2 d1] by key", docs, total)
	}

	// The empty path is the app root.
	root, total, err := store.ListFolder(ctx, testApp, "", 50, 0)
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	if total != 1 || root[0].DocumentID != "d4" {
		t.Errorf("root folder = %+v (total %d), want just d4", root, total)
	}

	// An unknown folder is an empty page, not an error.
	if _, total, err := store.ListFolder(ctx, testApp, "nowhere", 50, 0); err != nil || total != 0 {
		t.Errorf("unknown folder = %d,%v; want 0 matches", total, err)
	}
}

func TestSearch_FullText(t *testing.T) {
	store := searchReadsFixture(t)
	ctx := tenantCtx(testTenant)

	hits, total, err := store.Search(ctx, testApp, "install", 50, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || hits[0].DocumentID != "d1" {
		t.Errorf("search 'install' returned %d, want just d1 (tenant- and app-scoped)", total)
	}

	// "configure" matches d2's content.
	hits, _, err = store.Search(ctx, testApp, "configure", 50, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].DocumentID != "d2" {
		t.Errorf("search 'configure' did not match d2: %+v", hits)
	}
}

func TestSearch_SearchAll_CrossTenant(t *testing.T) {
	store := searchReadsFixture(t)

	// SearchAll is admin machinery: not tenant- or app-scoped, so "install"
	// matches d1 (testTenant/testApp), d5 (testTenant/otherApp), and d3
	// (otherTenant).
	hits, total, err := store.SearchAll(context.Background(), "install", 50, 0)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if total != 3 {
		t.Errorf("SearchAll 'install' returned %d, want 3 (all tenants and apps)", total)
	}
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h.DocumentID] = true
	}
	if !seen["d1"] || !seen["d3"] || !seen["d5"] {
		t.Errorf("SearchAll did not cross tenants/apps: %+v", hits)
	}
}
