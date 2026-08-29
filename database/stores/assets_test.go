//go:build testing

package stores

import (
	"bytes"
	"context"
	"errors"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub/mockdb"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

func assetCtx(tenant string) context.Context {
	return auth.WithPrincipal(context.Background(), auth.NewPrincipal("u-1", tenant, "", nil, nil))
}

// newAssetsTest builds an assets store over the mock bucket, with the apps
// store (which guards Put) over the mock SQL driver. Queue an appRow on cfg
// before each Put so the app-existence check passes; queue nothing to make it
// fail as an absent app.
func newAssetsTest(t *testing.T) (*Assets, *mockdb.Config) {
	t.Helper()
	sum.Reset()
	sum.New()
	db, _, cfg := mockdb.NewWithConfig()
	return NewAssets(testkit.NewBucketProvider(), NewApps(db, astqlpg.New())), cfg
}

// The full asset lifecycle: put, get (bytes + content type), list (metadata,
// prefix stripped), delete.
func TestAssets_PutGetListDelete(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")

	cfg.PushRowData(appRow()) // Put's app-existence guard
	meta, err := s.Put(ctx, "app-1", "images/logo.png", "image/png", []byte("PNGDATA"))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if meta.Key != "images/logo.png" || meta.ContentType != "image/png" || meta.Size != 7 {
		t.Errorf("put metadata = %+v", meta)
	}

	got, err := s.Get(ctx, "app-1", "images/logo.png")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got.Data, []byte("PNGDATA")) || got.ContentType != "image/png" {
		t.Errorf("get = {ct:%q data:%q}, want image/png/PNGDATA", got.ContentType, got.Data)
	}

	// List returns the user-facing key (tenant/app prefix stripped) and no bytes.
	list, err := s.List(ctx, "app-1", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "images/logo.png" || list[0].Data != nil {
		t.Errorf("list = %+v, want one metadata-only entry keyed images/logo.png", list)
	}

	if err := s.Delete(ctx, "app-1", "images/logo.png"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, "app-1", "images/logo.png"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}
}

// Put refuses an app the tenant does not have — writes create namespaces, and
// namespaces belong to real apps.
func TestAssets_PutMissingApp(t *testing.T) {
	s, _ := newAssetsTest(t) // no appRow queued: the app lookup finds nothing
	if _, err := s.Put(assetCtx("tenant-a"), "ghost", "k", "text/plain", []byte("x")); !errors.Is(err, ErrNotFound) {
		t.Errorf("put to missing app = %v, want ErrNotFound", err)
	}
}

// An empty content type is stored as octet-stream.
func TestAssets_DefaultsContentType(t *testing.T) {
	s, cfg := newAssetsTest(t)
	cfg.PushRowData(appRow())

	meta, err := s.Put(assetCtx("tenant-a"), "app-1", "blob", "", []byte("data"))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if meta.ContentType != "application/octet-stream" {
		t.Errorf("default content type = %q, want application/octet-stream", meta.ContentType)
	}
}

// Putting the same key overwrites the bytes — assets are not versioned.
func TestAssets_OverwriteSameKey(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")

	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "doc.pdf", "application/pdf", []byte("v1")); err != nil {
		t.Fatalf("put v1: %v", err)
	}
	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "doc.pdf", "text/plain", []byte("version two")); err != nil {
		t.Fatalf("put v2: %v", err)
	}

	got, err := s.Get(ctx, "app-1", "doc.pdf")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Data) != "version two" || got.ContentType != "text/plain" {
		t.Errorf("after overwrite = {ct:%q data:%q}, want the second put", got.ContentType, got.Data)
	}
	// Overwrite, not append: still exactly one asset.
	if list, _ := s.List(ctx, "app-1", ""); len(list) != 1 {
		t.Errorf("list = %d assets, want 1 (overwrite is destructive)", len(list))
	}
}

// The same key under two tenants addresses two independent assets; reads,
// lists, and deletes never cross the tenant boundary.
func TestAssets_TenantIsolation(t *testing.T) {
	s, cfg := newAssetsTest(t)
	a, b := assetCtx("tenant-a"), assetCtx("tenant-b")

	cfg.PushRowData(appRow())
	if _, err := s.Put(a, "app-1", "shared.txt", "text/plain", []byte("A's data")); err != nil {
		t.Fatalf("put a: %v", err)
	}
	cfg.PushRowData(appRow())
	if _, err := s.Put(b, "app-1", "shared.txt", "text/plain", []byte("B's data")); err != nil {
		t.Fatalf("put b: %v", err)
	}

	ga, err := s.Get(a, "app-1", "shared.txt")
	if err != nil {
		t.Fatalf("get a: %v", err)
	}
	gb, err := s.Get(b, "app-1", "shared.txt")
	if err != nil {
		t.Fatalf("get b: %v", err)
	}
	if string(ga.Data) != "A's data" || string(gb.Data) != "B's data" {
		t.Errorf("cross-tenant leak: a=%q b=%q", ga.Data, gb.Data)
	}

	// Each tenant lists only its own.
	if la, _ := s.List(a, "app-1", ""); len(la) != 1 || la[0].Key != "shared.txt" {
		t.Errorf("tenant a list = %+v, want just its own asset", la)
	}

	// A's delete leaves B untouched.
	if err := s.Delete(a, "app-1", "shared.txt"); err != nil {
		t.Fatalf("delete a: %v", err)
	}
	if _, err := s.Get(a, "app-1", "shared.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a still resolves its deleted asset: %v", err)
	}
	if _, err := s.Get(b, "app-1", "shared.txt"); err != nil {
		t.Errorf("b's asset was affected by a's delete: %v", err)
	}
}

// The same key under two apps of one tenant addresses two independent assets.
func TestAssets_AppIsolation(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")

	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "logo.png", "image/png", []byte("one")); err != nil {
		t.Fatalf("put app-1: %v", err)
	}
	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-2", "logo.png", "image/png", []byte("two")); err != nil {
		t.Fatalf("put app-2: %v", err)
	}

	g1, err := s.Get(ctx, "app-1", "logo.png")
	if err != nil {
		t.Fatalf("get app-1: %v", err)
	}
	g2, err := s.Get(ctx, "app-2", "logo.png")
	if err != nil {
		t.Fatalf("get app-2: %v", err)
	}
	if string(g1.Data) != "one" || string(g2.Data) != "two" {
		t.Errorf("cross-app leak: app-1=%q app-2=%q", g1.Data, g2.Data)
	}
	if l1, _ := s.List(ctx, "app-1", ""); len(l1) != 1 {
		t.Errorf("app-1 list = %d assets, want 1", len(l1))
	}
}

// A non-empty prefix narrows the listing to keys under it — the folder view.
func TestAssets_ListPrefix(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")

	for _, key := range []string{"images/logo.png", "images/icon.svg", "docs/spec.pdf"} {
		cfg.PushRowData(appRow())
		if _, err := s.Put(ctx, "app-1", key, "", []byte("x")); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	folder, err := s.List(ctx, "app-1", "images/")
	if err != nil {
		t.Fatalf("list prefix: %v", err)
	}
	if len(folder) != 2 {
		t.Errorf("images/ list = %+v, want the two image keys", folder)
	}
	for _, a := range folder {
		if a.Key != "images/logo.png" && a.Key != "images/icon.svg" {
			t.Errorf("unexpected key %q under images/", a.Key)
		}
	}
}

// Every operation refuses to run without a tenant.
func TestAssets_RequireTenant(t *testing.T) {
	s, _ := newAssetsTest(t)
	bg := context.Background()

	if _, err := s.Put(bg, "app-1", "k", "text/plain", []byte("x")); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("put without tenant = %v, want ErrNoTenant", err)
	}
	if _, err := s.Get(bg, "app-1", "k"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("get without tenant = %v, want ErrNoTenant", err)
	}
	if _, err := s.List(bg, "app-1", ""); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("list without tenant = %v, want ErrNoTenant", err)
	}
	if err := s.Delete(bg, "app-1", "k"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("delete without tenant = %v, want ErrNoTenant", err)
	}
}

// Deleting an absent key reports ErrNotFound.
func TestAssets_DeleteMissing(t *testing.T) {
	s, _ := newAssetsTest(t)
	if err := s.Delete(assetCtx("tenant-a"), "app-1", "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing = %v, want ErrNotFound", err)
	}
}
