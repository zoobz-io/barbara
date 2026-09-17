//go:build testing

package stores

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/grub/mockdb"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
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
	return NewAssets(testkit.NewBucketProvider(), NewApps(db, astqlpg.New()), db, astqlpg.New()), cfg
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

// folderRows is a queued asset_folders result: one row per (path, count,
// bytes, explicit) tuple, as the children query would return them.
func folderRows(rows ...[]any) *mockdb.RowData {
	return &mockdb.RowData{Columns: []string{"path", "count", "bytes", "explicit"}, Rows: rows}
}

// ListFolder assembles one level from the bucket's delimiter listing and the
// folder rows directly beneath it: direct assets by key, subfolders by name
// carrying their row's rollup, and nothing from other levels. The root and a
// nested folder are both levels.
func TestAssets_ListFolder(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")

	for _, key := range []string{
		"README.md",
		"images/logo.png",
		"images/icons/menu.svg",
		"images/icons/close.svg",
		"docs/spec.pdf",
	} {
		cfg.PushRowData(appRow())
		if _, err := s.Put(ctx, "app-1", key, "", []byte("x")); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	cfg.PushRowData(folderRows([]any{"docs", int64(1), int64(1), false}, []any{"images", int64(3), int64(3), false}))
	root, err := s.ListFolder(ctx, "app-1", "")
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	if root.Path != "" {
		t.Errorf("root path = %q, want empty", root.Path)
	}
	if len(root.Assets) != 1 || root.Assets[0].Key != "README.md" {
		t.Errorf("root assets = %+v, want README.md only", root.Assets)
	}
	wantFolders := []models.AssetFolder{{Name: "docs", Count: 1, Size: 1}, {Name: "images", Count: 3, Size: 3}}
	if !reflect.DeepEqual(root.Folders, wantFolders) {
		t.Errorf("root folders = %+v, want %+v", root.Folders, wantFolders)
	}

	// A nested level, addressed with a trailing slash, sees only its own
	// direct children.
	cfg.PushRowData(folderRows([]any{"images/icons", int64(2), int64(2), false}))
	images, err := s.ListFolder(ctx, "app-1", "images/")
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if images.Path != "images" {
		t.Errorf("images path = %q, want images", images.Path)
	}
	if len(images.Assets) != 1 || images.Assets[0].Key != "images/logo.png" {
		t.Errorf("images assets = %+v, want images/logo.png only", images.Assets)
	}
	if !reflect.DeepEqual(images.Folders, []models.AssetFolder{{Name: "icons", Count: 2, Size: 2}}) {
		t.Errorf("images folders = %+v, want icons(2, 2 B)", images.Folders)
	}

	// An empty level is empty slices, not nil, so it serializes as [].
	cfg.PushRowData(folderRows())
	empty, err := s.ListFolder(ctx, "app-1", "nothing")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if empty.Folders == nil || empty.Assets == nil || len(empty.Folders)+len(empty.Assets) != 0 {
		t.Errorf("empty level = %+v, want empty non-nil slices", empty)
	}
}

// The bucket decides which folders exist and the rows decorate them: a
// prefix the rows have not caught up with lists at a zero rollup, and an
// explicit row with nothing beneath it lists as an empty folder.
func TestAssets_ListFolder_RowsAndBucketDisagree(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")
	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "images/logo.png", "", []byte("x")); err != nil {
		t.Fatalf("put: %v", err)
	}

	cfg.PushRowData(folderRows([]any{"drafts", int64(0), int64(0), true}))
	root, err := s.ListFolder(ctx, "app-1", "")
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	want := []models.AssetFolder{{Name: "drafts"}, {Name: "images"}}
	if !reflect.DeepEqual(root.Folders, want) {
		t.Errorf("root folders = %+v, want %+v", root.Folders, want)
	}
}

// foldLevel strips the tenant/app scope from stored keys, infers a content
// type when the listing carries none, and dedupes a prefix the bucket
// reported on more than one page.
func TestFoldLevel(t *testing.T) {
	scope := "t/app/"
	objects := []grub.ObjectInfo{
		{Key: scope + "images/b.png", Size: 2},
		{Key: scope + "images/a.css", Size: 1, ContentType: "text/css"},
	}
	prefixes := []string{scope + "images/icons/", scope + "images/icons/", scope + "images/shots/"}
	written := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	rows := []*models.AssetFolderStat{{Path: "images/shots", Count: 4, Bytes: 40, LastWrittenAt: &written}}

	level := foldLevel("images", scope, objects, prefixes, rows)
	if level.Path != "images" {
		t.Errorf("path = %q, want images", level.Path)
	}
	if len(level.Assets) != 2 || level.Assets[0].Key != "images/a.css" || level.Assets[0].ContentType != "text/css" ||
		level.Assets[1].Key != "images/b.png" || level.Assets[1].ContentType != "image/png" {
		t.Errorf("assets = %+v", level.Assets)
	}
	want := []models.AssetFolder{{Name: "icons"}, {Name: "shots", Count: 4, Size: 40, LastWrittenAt: &written}}
	if !reflect.DeepEqual(level.Folders, want) {
		t.Errorf("folders = %+v, want %+v", level.Folders, want)
	}
}

// RebuildAssets pages through every app and rebuilds each from its own
// tenant: two apps on a short (last) page is two rebuilds, each stamping its
// own app. The fallback row answers every RETURNING upsert the rebuilds make.
func TestStores_RebuildAssets(t *testing.T) {
	sum.Reset()
	sum.New()
	db, capture, cfg := mockdb.NewWithConfig()
	cfg.SetRowData(&mockdb.RowData{Columns: []string{"id"}, Rows: [][]any{{"row-1"}}})
	st := New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())

	cfg.PushRowData(&mockdb.RowData{
		Columns: []string{"id", "tenant_id", "name"},
		Rows:    [][]any{{"app-1", "tenant-a", "one"}, {"app-2", "tenant-b", "two"}},
	})
	n, err := st.RebuildAssets(context.Background())
	if err != nil {
		t.Fatalf("RebuildAssets: %v", err)
	}
	if n != 2 {
		t.Errorf("apps rebuilt = %d, want 2", n)
	}
	var stamped []string
	for _, q := range capture.Queries {
		if strings.HasPrefix(q.Query, "INSERT") && strings.Contains(q.Query, `"asset_bookkeeping"`) {
			for _, a := range q.Args {
				if s, ok := a.(string); ok && strings.HasPrefix(s, "app-") {
					stamped = append(stamped, s)
				}
			}
		}
	}
	if !reflect.DeepEqual(stamped, []string{"app-1", "app-2"}) {
		t.Errorf("stamped apps = %v, want app-1 then app-2", stamped)
	}
}

// Move rewrites the object at the new key with its content type and drops
// the old one; the source must exist and the destination must be free.
func TestAssets_Move(t *testing.T) {
	s, cfg := newAssetsTest(t)
	ctx := assetCtx("tenant-a")
	fake, ok := s.bucket.(*testkit.BucketProvider)
	if !ok {
		t.Fatal("test store is not over the fake bucket")
	}
	for _, key := range []string{"images/logo.png", "images/taken.png"} {
		cfg.PushRowData(appRow())
		if _, err := s.Put(ctx, "app-1", key, "image/png", []byte("png")); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	moved, err := s.Move(ctx, "app-1", "images/logo.png", "/brand/logo-v2.png/")
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.Key != "brand/logo-v2.png" || moved.ContentType != "image/png" || moved.Size != 3 {
		t.Errorf("moved = %+v, want brand/logo-v2.png image/png 3 B", moved)
	}
	if _, ok := fake.Objects["tenant-a/app-1/images/logo.png"]; ok {
		t.Error("the source object is still in the bucket")
	}
	obj, ok := fake.Objects["tenant-a/app-1/brand/logo-v2.png"]
	if !ok || string(obj.Data) != "png" || obj.Info.ContentType != "image/png" {
		t.Errorf("destination object = %+v, want the bytes and content type carried over", obj)
	}

	if _, err := s.Move(ctx, "app-1", "images/nothing.png", "x.png"); !errors.Is(err, ErrNotFound) {
		t.Errorf("moving a missing key = %v, want ErrNotFound", err)
	}
	if _, err := s.Move(ctx, "app-1", "brand/logo-v2.png", "images/taken.png"); !errors.Is(err, ErrAssetExists) {
		t.Errorf("moving onto a taken key = %v, want ErrAssetExists", err)
	}
	if _, err := s.Move(ctx, "app-1", "brand/logo-v2.png", "brand/logo-v2.png"); !errors.Is(err, ErrAssetExists) {
		t.Errorf("moving onto itself = %v, want ErrAssetExists", err)
	}
	for _, bad := range []string{"", "a//b", "../x.png", "images/."} {
		if _, err := s.Move(ctx, "app-1", "brand/logo-v2.png", bad); !errors.Is(err, ErrInvalidAssetPath) {
			t.Errorf("moving to %q = %v, want ErrInvalidAssetPath", bad, err)
		}
	}
}

// CreateFolder rejects a path that is not plain segments before touching
// anything, and otherwise guards the app like a write.
func TestAssets_CreateFolder_Invalid(t *testing.T) {
	s, _ := newAssetsTest(t)
	ctx := assetCtx("tenant-a")
	for _, p := range []string{"", "/", "a//b", "./a", "a/../b", "a/."} {
		if _, err := s.CreateFolder(ctx, "app-1", p); !errors.Is(err, ErrInvalidAssetPath) {
			t.Errorf("CreateFolder(%q) = %v, want ErrInvalidAssetPath", p, err)
		}
	}
	// No app row queued: the guard fails as an absent app.
	if _, err := s.CreateFolder(ctx, "app-1", "images"); !errors.Is(err, ErrNotFound) {
		t.Errorf("CreateFolder for an absent app = %v, want ErrNotFound", err)
	}
}

// Listing falls back to the key's extension when the bucket listing omits a
// content type — S3 listings carry none. Only extensions in Go's builtin mime
// table or the store's own registrations appear here, so the test is
// environment-independent.
func TestContentTypeForKey(t *testing.T) {
	cases := map[string]string{
		"images/logo.png": "image/png",
		"styles.css":      "text/css",
		"docs/spec.pdf":   "application/pdf",
		"README.md":       "text/markdown",
		"robots.txt":      "text/plain",
		"data/prices.csv": "text/csv",
		"fonts/ui.woff2":  "font/woff2",
		"blob":            "application/octet-stream",
	}
	for key, want := range cases {
		if got := contentTypeForKey(key); got != want {
			t.Errorf("contentTypeForKey(%q) = %q, want %q", key, got, want)
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
