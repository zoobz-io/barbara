//go:build testing

package stores

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

func assetCtx(tenant string) context.Context {
	return auth.WithPrincipal(context.Background(), auth.NewPrincipal("u-1", tenant, "", nil, nil))
}

// The full asset lifecycle: put, get (bytes + content type), list (metadata,
// prefix stripped), delete.
func TestAssets_PutGetListDelete(t *testing.T) {
	s := NewAssets(testkit.NewBucketProvider())
	ctx := assetCtx("tenant-a")

	meta, err := s.Put(ctx, "images/logo.png", "image/png", []byte("PNGDATA"))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if meta.Key != "images/logo.png" || meta.ContentType != "image/png" || meta.Size != 7 {
		t.Errorf("put metadata = %+v", meta)
	}

	got, err := s.Get(ctx, "images/logo.png")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got.Data, []byte("PNGDATA")) || got.ContentType != "image/png" {
		t.Errorf("get = {ct:%q data:%q}, want image/png/PNGDATA", got.ContentType, got.Data)
	}

	// List returns the user-facing key (tenant prefix stripped) and no bytes.
	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "images/logo.png" || list[0].Data != nil {
		t.Errorf("list = %+v, want one metadata-only entry keyed images/logo.png", list)
	}

	if err := s.Delete(ctx, "images/logo.png"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, "images/logo.png"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}
}

// An empty content type is stored as octet-stream.
func TestAssets_DefaultsContentType(t *testing.T) {
	s := NewAssets(testkit.NewBucketProvider())
	ctx := assetCtx("tenant-a")

	meta, err := s.Put(ctx, "blob", "", []byte("data"))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if meta.ContentType != "application/octet-stream" {
		t.Errorf("default content type = %q, want application/octet-stream", meta.ContentType)
	}
}

// Putting the same key overwrites the bytes — assets are not versioned.
func TestAssets_OverwriteSameKey(t *testing.T) {
	s := NewAssets(testkit.NewBucketProvider())
	ctx := assetCtx("tenant-a")

	if _, err := s.Put(ctx, "doc.pdf", "application/pdf", []byte("v1")); err != nil {
		t.Fatalf("put v1: %v", err)
	}
	if _, err := s.Put(ctx, "doc.pdf", "text/plain", []byte("version two")); err != nil {
		t.Fatalf("put v2: %v", err)
	}

	got, err := s.Get(ctx, "doc.pdf")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Data) != "version two" || got.ContentType != "text/plain" {
		t.Errorf("after overwrite = {ct:%q data:%q}, want the second put", got.ContentType, got.Data)
	}
	// Overwrite, not append: still exactly one asset.
	if list, _ := s.List(ctx); len(list) != 1 {
		t.Errorf("list = %d assets, want 1 (overwrite is destructive)", len(list))
	}
}

// The same key under two tenants addresses two independent assets; reads, lists,
// and deletes never cross the tenant boundary.
func TestAssets_TenantIsolation(t *testing.T) {
	s := NewAssets(testkit.NewBucketProvider())
	a, b := assetCtx("tenant-a"), assetCtx("tenant-b")

	if _, err := s.Put(a, "shared.txt", "text/plain", []byte("A's data")); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if _, err := s.Put(b, "shared.txt", "text/plain", []byte("B's data")); err != nil {
		t.Fatalf("put b: %v", err)
	}

	ga, err := s.Get(a, "shared.txt")
	if err != nil {
		t.Fatalf("get a: %v", err)
	}
	gb, err := s.Get(b, "shared.txt")
	if err != nil {
		t.Fatalf("get b: %v", err)
	}
	if string(ga.Data) != "A's data" || string(gb.Data) != "B's data" {
		t.Errorf("cross-tenant leak: a=%q b=%q", ga.Data, gb.Data)
	}

	// Each tenant lists only its own.
	if la, _ := s.List(a); len(la) != 1 || la[0].Key != "shared.txt" {
		t.Errorf("tenant a list = %+v, want just its own asset", la)
	}

	// A's delete leaves B untouched.
	if err := s.Delete(a, "shared.txt"); err != nil {
		t.Fatalf("delete a: %v", err)
	}
	if _, err := s.Get(a, "shared.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a still resolves its deleted asset: %v", err)
	}
	if _, err := s.Get(b, "shared.txt"); err != nil {
		t.Errorf("b's asset was affected by a's delete: %v", err)
	}
}

// Every operation refuses to run without a tenant.
func TestAssets_RequireTenant(t *testing.T) {
	s := NewAssets(testkit.NewBucketProvider())
	bg := context.Background()

	if _, err := s.Put(bg, "k", "text/plain", []byte("x")); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("put without tenant = %v, want ErrNoTenant", err)
	}
	if _, err := s.Get(bg, "k"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("get without tenant = %v, want ErrNoTenant", err)
	}
	if _, err := s.List(bg); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("list without tenant = %v, want ErrNoTenant", err)
	}
	if err := s.Delete(bg, "k"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("delete without tenant = %v, want ErrNoTenant", err)
	}
}

// Deleting an absent key reports ErrNotFound.
func TestAssets_DeleteMissing(t *testing.T) {
	s := NewAssets(testkit.NewBucketProvider())
	if err := s.Delete(assetCtx("tenant-a"), "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing = %v, want ErrNotFound", err)
	}
}
