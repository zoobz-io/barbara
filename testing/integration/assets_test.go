//go:build testing

package integration

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	minio "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub"
	grubminio "github.com/zoobz-io/grub/minio"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
)

// minioBucket builds the real MinIO-backed bucket provider for integration
// tests, ensuring the bucket exists. It skips when MinIO is unreachable (and, in
// CI, hard-fails via integrationSkip since the stack is provisioned there).
func minioBucket(t *testing.T) grub.BucketProvider {
	t.Helper()
	endpoint := env("APP_STORAGE_ENDPOINT", "localhost:19000")
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
	t.Cleanup(func() { resetDB(t, db); _ = db.Close() })
	apps := stores.NewApps(db, astqlpg.New())
	s := stores.NewAssets(minioBucket(t), apps, db, astqlpg.New())

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

// TestAssets_Bookkeeping proves the bookkeeping against real Postgres and
// MinIO: writes and deletes keep the folder rollups, kind breakdown, and daily
// series exact, and a rebuild from the bucket lands on the same numbers.
func TestAssets_Bookkeeping(t *testing.T) {
	db := pgDB(t)
	t.Cleanup(func() { resetDB(t, db); _ = db.Close() })
	apps := stores.NewApps(db, astqlpg.New())
	s := stores.NewAssets(minioBucket(t), apps, db, astqlpg.New())
	ctx := tenantCtx(uuid.NewString())
	app, err := apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("creating app: %v", err)
	}

	for key, ct := range map[string]string{
		"README.md":             "text/markdown",
		"images/logo.png":       "image/png",
		"images/icons/menu.svg": "image/svg+xml",
	} {
		if _, err := s.Put(ctx, app.ID, key, ct, []byte(strings.Repeat("x", 10))); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	stats, err := s.Stats(ctx, app.ID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Root.Count != 3 || stats.Root.Bytes != 30 || stats.Root.LastWrittenAt == nil {
		t.Errorf("root after 3 puts = %+v, want 3 objects / 30 bytes / last written set", stats.Root)
	}
	kinds := map[string][2]int64{}
	for _, k := range stats.Kinds {
		kinds[string(k.Kind)] = [2]int64{k.Count, k.Bytes}
	}
	if !reflect.DeepEqual(kinds, map[string][2]int64{"image": {2, 20}, "text": {1, 10}}) {
		t.Errorf("kinds = %v, want image 2/20 and text 1/10", kinds)
	}
	var todayAll int64
	for _, d := range stats.Days {
		if d.Kind == "" {
			todayAll += d.Count
		}
	}
	if todayAll != 3 {
		t.Errorf("all-kinds day rows sum to %d objects, want 3", todayAll)
	}
	if stats.ComputedAt != nil {
		t.Errorf("computed_at = %v before any rebuild, want nil", stats.ComputedAt)
	}

	// Overwrite is a size change, not a fourth asset; delete removes one.
	if _, err := s.Put(ctx, app.ID, "README.md", "text/markdown", []byte("x")); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if err := s.Delete(ctx, app.ID, "images/icons/menu.svg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	live, err := s.Stats(ctx, app.ID)
	if err != nil {
		t.Fatalf("stats after changes: %v", err)
	}
	if live.Root.Count != 2 || live.Root.Bytes != 11 {
		t.Errorf("root after overwrite+delete = %d objects / %d bytes, want 2 / 11", live.Root.Count, live.Root.Bytes)
	}

	// The rebuild lands on the same numbers and stamps the app.
	if err := s.Rebuild(ctx, app.ID); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	rebuilt, err := s.Stats(ctx, app.ID)
	if err != nil {
		t.Fatalf("stats after rebuild: %v", err)
	}
	if rebuilt.Root.Count != live.Root.Count || rebuilt.Root.Bytes != live.Root.Bytes {
		t.Errorf("rebuild root = %d / %d, live was %d / %d", rebuilt.Root.Count, rebuilt.Root.Bytes, live.Root.Count, live.Root.Bytes)
	}
	if len(rebuilt.Kinds) != len(live.Kinds) {
		t.Errorf("rebuild kinds = %d rows, live had %d", len(rebuilt.Kinds), len(live.Kinds))
	}
	for i := range live.Kinds {
		if i < len(rebuilt.Kinds) && (rebuilt.Kinds[i].Count != live.Kinds[i].Count || rebuilt.Kinds[i].Bytes != live.Kinds[i].Bytes) {
			t.Errorf("rebuild kind %s = %d / %d, live was %d / %d", live.Kinds[i].Kind,
				rebuilt.Kinds[i].Count, rebuilt.Kinds[i].Bytes, live.Kinds[i].Count, live.Kinds[i].Bytes)
		}
	}
	if rebuilt.ComputedAt == nil {
		t.Error("computed_at still nil after rebuild")
	}
}

// TestAssets_Folders proves the folder view against real MinIO and Postgres:
// a level comes from the bucket's delimiter listing plus the folder rows, so
// subfolders carry their rollup, an explicit folder lists before it holds
// anything, and its ancestors list with it.
func TestAssets_Folders(t *testing.T) {
	db := pgDB(t)
	t.Cleanup(func() { resetDB(t, db); _ = db.Close() })
	apps := stores.NewApps(db, astqlpg.New())
	s := stores.NewAssets(minioBucket(t), apps, db, astqlpg.New())
	ctx := tenantCtx(uuid.NewString())
	app, err := apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("creating app: %v", err)
	}

	for key, size := range map[string]int{
		"README.md":              1,
		"images/logo.png":        10,
		"images/icons/menu.svg":  100,
		"images/icons/close.svg": 1000,
	} {
		if _, err := s.Put(ctx, app.ID, key, "", bytes.Repeat([]byte("x"), size)); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	root, err := s.ListFolder(ctx, app.ID, "")
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	if len(root.Assets) != 1 || root.Assets[0].Key != "README.md" || root.Assets[0].ContentType != "text/markdown" {
		t.Errorf("root assets = %+v, want README.md (text/markdown)", root.Assets)
	}
	if !reflect.DeepEqual(rollups(root.Folders), []models.AssetFolder{{Name: "images", Count: 3, Size: 1110}}) {
		t.Errorf("root folders = %+v, want images 3 / 1110 B", root.Folders)
	}
	if root.Folders[0].LastWrittenAt == nil {
		t.Error("images folder carries no last-written time")
	}
	images, err := s.ListFolder(ctx, app.ID, "images")
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if len(images.Assets) != 1 || images.Assets[0].Key != "images/logo.png" {
		t.Errorf("images assets = %+v, want images/logo.png only", images.Assets)
	}
	if !reflect.DeepEqual(rollups(images.Folders), []models.AssetFolder{{Name: "icons", Count: 2, Size: 1100}}) {
		t.Errorf("images folders = %+v, want icons 2 / 1100 B", images.Folders)
	}

	// An explicit folder two levels down lists empty, and brings its parent
	// into the root listing.
	created, err := s.CreateFolder(ctx, app.ID, "docs/drafts")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if created.Path != "docs/drafts" || len(created.Folders)+len(created.Assets) != 0 {
		t.Errorf("created level = %+v, want empty docs/drafts", created)
	}
	root, err = s.ListFolder(ctx, app.ID, "")
	if err != nil {
		t.Fatalf("list root again: %v", err)
	}
	if !reflect.DeepEqual(rollups(root.Folders), []models.AssetFolder{{Name: "docs"}, {Name: "images", Count: 3, Size: 1110}}) {
		t.Errorf("root folders after create = %+v, want docs (empty) and images", root.Folders)
	}
	docs, err := s.ListFolder(ctx, app.ID, "docs")
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if !reflect.DeepEqual(rollups(docs.Folders), []models.AssetFolder{{Name: "drafts"}}) {
		t.Errorf("docs folders = %+v, want drafts (empty)", docs.Folders)
	}

	// The explicit folder survives an upload and a delete beneath it, and a
	// rebuild.
	if _, err := s.Put(ctx, app.ID, "docs/drafts/a.txt", "text/plain", []byte("a")); err != nil {
		t.Fatalf("put into explicit folder: %v", err)
	}
	if err := s.Delete(ctx, app.ID, "docs/drafts/a.txt"); err != nil {
		t.Fatalf("delete from explicit folder: %v", err)
	}
	if err := s.Rebuild(ctx, app.ID); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	docs, err = s.ListFolder(ctx, app.ID, "docs")
	if err != nil {
		t.Fatalf("list docs after rebuild: %v", err)
	}
	if !reflect.DeepEqual(rollups(docs.Folders), []models.AssetFolder{{Name: "drafts"}}) {
		t.Errorf("docs folders after empty+rebuild = %+v, want drafts still listed", docs.Folders)
	}
}

// TestAssets_Move proves a move against real MinIO and Postgres: the bytes
// and content type land at the new key, the old key is gone, and the folder
// rollups move with the object in one step.
func TestAssets_Move(t *testing.T) {
	db := pgDB(t)
	t.Cleanup(func() { resetDB(t, db); _ = db.Close() })
	apps := stores.NewApps(db, astqlpg.New())
	s := stores.NewAssets(minioBucket(t), apps, db, astqlpg.New())
	ctx := tenantCtx(uuid.NewString())
	app, err := apps.Create(ctx, "site")
	if err != nil {
		t.Fatalf("creating app: %v", err)
	}
	if _, err := s.Put(ctx, app.ID, "images/logo.png", "image/png", []byte("png!")); err != nil {
		t.Fatalf("put: %v", err)
	}

	moved, err := s.Move(ctx, app.ID, "images/logo.png", "brand/logo.png")
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.Key != "brand/logo.png" || moved.ContentType != "image/png" || moved.Size != 4 {
		t.Errorf("moved = %+v, want brand/logo.png image/png 4 B", moved)
	}
	got, err := s.Get(ctx, app.ID, "brand/logo.png")
	if err != nil || string(got.Data) != "png!" || got.ContentType != "image/png" {
		t.Errorf("get moved = %+v, %v; want the bytes and content type", got, err)
	}
	if _, err := s.Get(ctx, app.ID, "images/logo.png"); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("get source after move = %v, want ErrNotFound", err)
	}

	root, err := s.ListFolder(ctx, app.ID, "")
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	if !reflect.DeepEqual(names(root.Folders), []string{"brand"}) || root.Folders[0].Count != 1 || root.Folders[0].Size != 4 {
		t.Errorf("root folders after move = %+v, want brand alone at 1 / 4 B", root.Folders)
	}
	stats, err := s.Stats(ctx, app.ID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Root.Count != 1 || stats.Root.Bytes != 4 {
		t.Errorf("root rollup after move = %d / %d, want 1 / 4", stats.Root.Count, stats.Root.Bytes)
	}
}

// rollups strips the last-written times, which vary run to run, so a level's
// folders compare on name, count, and size.
func rollups(folders []models.AssetFolder) []models.AssetFolder {
	out := make([]models.AssetFolder, len(folders))
	for i, f := range folders {
		out[i] = models.AssetFolder{Name: f.Name, Count: f.Count, Size: f.Size}
	}
	return out
}

// names lists folders by name.
func names(folders []models.AssetFolder) []string {
	out := make([]string, 0, len(folders))
	for _, f := range folders {
		out = append(out, f.Name)
	}
	return out
}
