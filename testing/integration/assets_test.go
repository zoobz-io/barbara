//go:build testing

package integration

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	minio "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
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

// TestAssets_MinIO exercises the asset store against real object storage,
// proving the two invariants the domain rests on: putting the same key
// overwrites (assets are not versioned), and tenants are isolated even though
// they share one bucket.
func TestAssets_MinIO(t *testing.T) {
	s := stores.NewAssets(minioBucket(t))
	a := tenantCtx(testTenant)
	b := tenantCtx(otherTenant)

	// Clean slate + cleanup, so the run is idempotent against a persistent bucket.
	keys := []string{"reports/q3.pdf", "shared/logo.png"}
	cleanup := func() {
		for _, k := range keys {
			_ = s.Delete(a, k)
			_ = s.Delete(b, k)
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	// Overwrite: the second put wins, and there is still just one object.
	if _, err := s.Put(a, "reports/q3.pdf", "application/pdf", []byte("draft")); err != nil {
		t.Fatalf("put v1: %v", err)
	}
	if _, err := s.Put(a, "reports/q3.pdf", "application/pdf", []byte("final revision")); err != nil {
		t.Fatalf("put v2: %v", err)
	}
	got, err := s.Get(a, "reports/q3.pdf")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got.Data, []byte("final revision")) {
		t.Errorf("overwrite: got %q, want the second put", got.Data)
	}

	// Tenant isolation: the same key under two tenants is two independent objects.
	if _, err := s.Put(a, "shared/logo.png", "image/png", []byte("A's logo")); err != nil {
		t.Fatalf("put a shared: %v", err)
	}
	if _, err := s.Put(b, "shared/logo.png", "image/png", []byte("B's logo")); err != nil {
		t.Fatalf("put b shared: %v", err)
	}
	ga, err := s.Get(a, "shared/logo.png")
	if err != nil {
		t.Fatalf("get a shared: %v", err)
	}
	gb, err := s.Get(b, "shared/logo.png")
	if err != nil {
		t.Fatalf("get b shared: %v", err)
	}
	if string(ga.Data) != "A's logo" || string(gb.Data) != "B's logo" {
		t.Errorf("cross-tenant leak: a=%q b=%q", ga.Data, gb.Data)
	}

	// A's listing sees only A's objects (the two keys A put), never B's.
	listA, err := s.List(a)
	if err != nil {
		t.Fatalf("list a: %v", err)
	}
	seen := map[string]bool{}
	for _, as := range listA {
		seen[as.Key] = true
	}
	if !seen["reports/q3.pdf"] || !seen["shared/logo.png"] || len(listA) != 2 {
		t.Errorf("tenant a list = %+v, want exactly its own two keys", listA)
	}

	// A's delete removes only A's object; B's remains.
	if err := s.Delete(a, "shared/logo.png"); err != nil {
		t.Fatalf("delete a shared: %v", err)
	}
	if _, err := s.Get(a, "shared/logo.png"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("a still resolves its deleted asset: %v", err)
	}
	if _, err := s.Get(b, "shared/logo.png"); err != nil {
		t.Errorf("b's asset was affected by a's delete: %v", err)
	}
}
