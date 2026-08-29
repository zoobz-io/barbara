//go:build testing

package integration

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	minio "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub"
	grubminio "github.com/zoobz-io/grub/minio"

	"github.com/zoobz-io/barbara/database/stores"
)

// minioBucket builds the real MinIO-backed bucket provider for integration
// tests, ensuring the bucket exists. It skips when MinIO is unreachable (and, in
// CI, hard-fails via integrationSkip since the stack is provisioned there).
func minioBucket(t *testing.T) grub.BucketProvider {
	t.Helper()
	endpoint := env("APP_STORAGE_ENDPOINT", "localhost:9000")
	bucket := env("APP_STORAGE_BUCKET", "barbara")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  miniocreds.NewStaticV4(env("APP_STORAGE_ACCESS_KEY", "minioadmin"), env("APP_STORAGE_SECRET_KEY", "minioadmin"), ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("minio client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		integrationSkip(t, "MinIO not reachable at %s (%v)", endpoint, err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("make bucket %q: %v", bucket, err)
		}
	}
	return grubminio.New(client, bucket)
}

// TestAssets_MinIO exercises the asset store against real object storage and
// Postgres (the apps guard), proving the invariants the domain rests on:
// putting the same key overwrites (assets are not versioned), tenants and apps
// are isolated even though they share one bucket, a prefix narrows a listing
// to the folder view, and a write to a nonexistent app is refused.
func TestAssets_MinIO(t *testing.T) {
	db := pgDB(t)
	defer func() { _ = db.Close() }()
	apps := stores.NewApps(db, astqlpg.New())
	s := stores.NewAssets(minioBucket(t), apps)

	// Fresh tenants per run, so the test is self-contained against persistent
	// backing services.
	a := tenantCtx(uuid.NewString())
	b := tenantCtx(uuid.NewString())

	appA1, err := apps.Create(a, "site")
	if err != nil {
		t.Fatalf("creating app a1: %v", err)
	}
	appA2, err := apps.Create(a, "blog")
	if err != nil {
		t.Fatalf("creating app a2: %v", err)
	}
	appB, err := apps.Create(b, "site")
	if err != nil {
		t.Fatalf("creating app b: %v", err)
	}

	t.Cleanup(func() {
		for _, k := range []string{"reports/q3.pdf", "shared/logo.png", "images/icon.svg"} {
			_ = s.Delete(a, appA1.ID, k)
			_ = s.Delete(a, appA2.ID, k)
			_ = s.Delete(b, appB.ID, k)
		}
	})

	// A write to an app the tenant does not have is refused.
	if _, err := s.Put(a, uuid.NewString(), "k.txt", "text/plain", []byte("x")); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("put to missing app = %v, want ErrNotFound", err)
	}
	// Including another tenant's app.
	if _, err := s.Put(a, appB.ID, "k.txt", "text/plain", []byte("x")); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("put to another tenant's app = %v, want ErrNotFound", err)
	}

	// Overwrite: the second put wins, and there is still just one object.
	if _, err := s.Put(a, appA1.ID, "reports/q3.pdf", "application/pdf", []byte("draft")); err != nil {
		t.Fatalf("put v1: %v", err)
	}
	if _, err := s.Put(a, appA1.ID, "reports/q3.pdf", "application/pdf", []byte("final revision")); err != nil {
		t.Fatalf("put v2: %v", err)
	}
	got, err := s.Get(a, appA1.ID, "reports/q3.pdf")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got.Data, []byte("final revision")) {
		t.Errorf("overwrite: got %q, want the second put", got.Data)
	}

	// Tenant and app isolation: the same key under two tenants — and under two
	// apps of one tenant — addresses independent objects.
	if _, err := s.Put(a, appA1.ID, "shared/logo.png", "image/png", []byte("A1's logo")); err != nil {
		t.Fatalf("put a1 shared: %v", err)
	}
	if _, err := s.Put(a, appA2.ID, "shared/logo.png", "image/png", []byte("A2's logo")); err != nil {
		t.Fatalf("put a2 shared: %v", err)
	}
	if _, err := s.Put(b, appB.ID, "shared/logo.png", "image/png", []byte("B's logo")); err != nil {
		t.Fatalf("put b shared: %v", err)
	}
	g1, err := s.Get(a, appA1.ID, "shared/logo.png")
	if err != nil {
		t.Fatalf("get a1 shared: %v", err)
	}
	g2, err := s.Get(a, appA2.ID, "shared/logo.png")
	if err != nil {
		t.Fatalf("get a2 shared: %v", err)
	}
	gb, err := s.Get(b, appB.ID, "shared/logo.png")
	if err != nil {
		t.Fatalf("get b shared: %v", err)
	}
	if string(g1.Data) != "A1's logo" || string(g2.Data) != "A2's logo" || string(gb.Data) != "B's logo" {
		t.Errorf("isolation leak: a1=%q a2=%q b=%q", g1.Data, g2.Data, gb.Data)
	}

	// The app listing sees only its own objects; a prefix narrows to the folder.
	if _, err := s.Put(a, appA1.ID, "images/icon.svg", "image/svg+xml", []byte("<svg/>")); err != nil {
		t.Fatalf("put icon: %v", err)
	}
	listA1, err := s.List(a, appA1.ID, "")
	if err != nil {
		t.Fatalf("list a1: %v", err)
	}
	if len(listA1) != 3 {
		t.Errorf("app a1 list = %+v, want exactly its own three keys", listA1)
	}
	folder, err := s.List(a, appA1.ID, "images/")
	if err != nil {
		t.Fatalf("list a1 images/: %v", err)
	}
	if len(folder) != 1 || folder[0].Key != "images/icon.svg" {
		t.Errorf("images/ folder = %+v, want just the icon", folder)
	}

	// A1's delete removes only A1's object; the same key elsewhere remains.
	if err := s.Delete(a, appA1.ID, "shared/logo.png"); err != nil {
		t.Fatalf("delete a1 shared: %v", err)
	}
	if _, err := s.Get(a, appA1.ID, "shared/logo.png"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("a1 still resolves its deleted asset: %v", err)
	}
	if _, err := s.Get(a, appA2.ID, "shared/logo.png"); err != nil {
		t.Errorf("a2's asset was affected by a1's delete: %v", err)
	}
	if _, err := s.Get(b, appB.ID, "shared/logo.png"); err != nil {
		t.Errorf("b's asset was affected by a1's delete: %v", err)
	}
}
