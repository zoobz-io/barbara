//go:build testing

package stores

import (
	"context"
	"errors"
	"fmt"
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

var (
	errBucket = errors.New("bucket down")
	errDB     = errors.New("db down")
)

// faultyBucket is the in-memory bucket with one operation broken at a time.
// statFailAt breaks only the n-th Stat (1-based, counted from the last
// reset), so a move can fail on its destination check after its source
// check succeeded.
type faultyBucket struct {
	*testkit.BucketProvider
	statErr, putErr, getErr, deleteErr, listErr, levelErr error
	statFailAt, stats                                     int
}

func (b *faultyBucket) Stat(ctx context.Context, key string) (*grub.ObjectInfo, error) {
	b.stats++
	if b.statErr != nil && (b.statFailAt == 0 || b.stats == b.statFailAt) {
		return nil, b.statErr
	}
	return b.BucketProvider.Stat(ctx, key)
}

func (b *faultyBucket) Put(ctx context.Context, key string, data []byte, info *grub.ObjectInfo) error {
	if b.putErr != nil {
		return b.putErr
	}
	return b.BucketProvider.Put(ctx, key, data, info)
}

func (b *faultyBucket) Get(ctx context.Context, key string) ([]byte, *grub.ObjectInfo, error) {
	if b.getErr != nil {
		return nil, nil, b.getErr
	}
	return b.BucketProvider.Get(ctx, key)
}

func (b *faultyBucket) Delete(ctx context.Context, key string) error {
	if b.deleteErr != nil {
		return b.deleteErr
	}
	return b.BucketProvider.Delete(ctx, key)
}

func (b *faultyBucket) List(ctx context.Context, prefix string, limit int) ([]grub.ObjectInfo, error) {
	if b.listErr != nil {
		return nil, b.listErr
	}
	return b.BucketProvider.List(ctx, prefix, limit)
}

func (b *faultyBucket) ListLevel(ctx context.Context, prefix, delimiter, cursor string, limit int) (*grub.Level, error) {
	if b.levelErr != nil {
		return nil, b.levelErr
	}
	return b.BucketProvider.ListLevel(ctx, prefix, delimiter, cursor, limit)
}

// newFaultyAssetsTest is newAssetsTest over a faultyBucket, with every SQL
// read answering one row so the app-existence check passes.
func newFaultyAssetsTest(t *testing.T) (*Assets, *faultyBucket, *mockdb.Config) {
	t.Helper()
	sum.Reset()
	sum.New()
	db, _, cfg := mockdb.NewWithConfig()
	cfg.SetRowData(&mockdb.RowData{Columns: []string{"id"}, Rows: [][]any{{"row-1"}}})
	bucket := &faultyBucket{BucketProvider: testkit.NewBucketProvider()}
	return NewAssets(bucket, NewApps(db, astqlpg.New()), db, astqlpg.New()), bucket, cfg
}

// Every bucket operation the store makes surfaces its failure, naming the
// step and the key and wrapping the bucket's error.
func TestAssets_BucketFailures(t *testing.T) {
	ctx := assetCtx("tenant-a")
	cases := []struct {
		name    string
		arrange func(b *faultyBucket)
		act     func(s *Assets) error
		want    string
	}{
		{"put stat", func(b *faultyBucket) { b.statErr = errBucket },
			func(s *Assets) error { _, err := s.Put(ctx, "app-1", "k", "text/plain", []byte("x")); return err },
			`checking asset "k"`},
		{"put write", func(b *faultyBucket) { b.putErr = errBucket },
			func(s *Assets) error { _, err := s.Put(ctx, "app-1", "k", "text/plain", []byte("x")); return err },
			`putting asset "k"`},
		{"get", func(b *faultyBucket) { b.getErr = errBucket },
			func(s *Assets) error { _, err := s.Get(ctx, "app-1", "images/logo.png"); return err },
			`getting asset "images/logo.png"`},
		{"list", func(b *faultyBucket) { b.listErr = errBucket },
			func(s *Assets) error { _, err := s.List(ctx, "app-1", ""); return err },
			"listing assets"},
		{"list folder", func(b *faultyBucket) { b.levelErr = errBucket },
			func(s *Assets) error { _, err := s.ListFolder(ctx, "app-1", "images"); return err },
			`listing asset folder "images"`},
		{"rebuild listing", func(b *faultyBucket) { b.listErr = errBucket },
			func(s *Assets) error { return s.Rebuild(ctx, "app-1") },
			"listing assets"},
		{"move source stat", func(b *faultyBucket) { b.statErr = errBucket },
			func(s *Assets) error {
				_, err := s.Move(ctx, "app-1", "images/logo.png", "images/brand.png")
				return err
			},
			`checking asset "images/logo.png"`},
		{"move destination stat", func(b *faultyBucket) { b.stats, b.statErr, b.statFailAt = 0, errBucket, 2 },
			func(s *Assets) error {
				_, err := s.Move(ctx, "app-1", "images/logo.png", "images/brand.png")
				return err
			},
			`checking asset "images/brand.png"`},
		{"move read", func(b *faultyBucket) { b.getErr = errBucket },
			func(s *Assets) error {
				_, err := s.Move(ctx, "app-1", "images/logo.png", "images/brand.png")
				return err
			},
			`reading asset "images/logo.png"`},
		{"move write", func(b *faultyBucket) { b.putErr = errBucket },
			func(s *Assets) error {
				_, err := s.Move(ctx, "app-1", "images/logo.png", "images/brand.png")
				return err
			},
			`writing asset "images/brand.png"`},
		{"move remove", func(b *faultyBucket) { b.deleteErr = errBucket },
			func(s *Assets) error {
				_, err := s.Move(ctx, "app-1", "images/logo.png", "images/brand.png")
				return err
			},
			`removing moved asset "images/logo.png"`},
		{"delete stat", func(b *faultyBucket) { b.statErr = errBucket },
			func(s *Assets) error { return s.Delete(ctx, "app-1", "images/logo.png") },
			`checking asset "images/logo.png"`},
		{"delete remove", func(b *faultyBucket) { b.deleteErr = errBucket },
			func(s *Assets) error { return s.Delete(ctx, "app-1", "images/logo.png") },
			`deleting asset "images/logo.png"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, bucket, _ := newFaultyAssetsTest(t)
			if _, err := s.Put(ctx, "app-1", "images/logo.png", "image/png", []byte("png")); err != nil {
				t.Fatalf("seeding asset: %v", err)
			}
			tc.arrange(bucket)
			err := tc.act(s)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !errors.Is(err, errBucket) {
				t.Errorf("error = %v, want one mentioning %q and wrapping the bucket error", err, tc.want)
			}
		})
	}
}

// An object that vanishes between the existence check and the delete is
// still reported as not found.
func TestAssets_DeleteVanished(t *testing.T) {
	ctx := assetCtx("tenant-a")
	s, bucket, _ := newFaultyAssetsTest(t)
	if _, err := s.Put(ctx, "app-1", "a.txt", "text/plain", []byte("x")); err != nil {
		t.Fatal(err)
	}
	bucket.deleteErr = grub.ErrNotFound
	if err := s.Delete(ctx, "app-1", "a.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete of a vanished object = %v, want ErrNotFound", err)
	}
}

// The tenant-scoped reads and writes all refuse a context without one.
func TestAssets_RequireTenant_All(t *testing.T) {
	s, _, _ := newFaultyAssetsTest(t)
	bg := context.Background()
	calls := map[string]func() error{
		"ListFolder":   func() error { _, err := s.ListFolder(bg, "app-1", ""); return err },
		"CreateFolder": func() error { _, err := s.CreateFolder(bg, "app-1", "images"); return err },
		"Stats":        func() error { _, err := s.Stats(bg, "app-1"); return err },
		"Move":         func() error { _, err := s.Move(bg, "app-1", "a", "b"); return err },
		"Rebuild":      func() error { return s.Rebuild(bg, "app-1") },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, auth.ErrNoTenant) {
			t.Errorf("%s without tenant = %v, want ErrNoTenant", name, err)
		}
	}
}

// A bookkeeping failure after a successful bucket write is reported, not
// returned: the write still succeeds and the object is the truth.
func TestAssets_BookkeepingFailureDoesNotFailTheWrite(t *testing.T) {
	ctx := assetCtx("tenant-a")
	s, bucket, cfg := newFaultyAssetsTest(t)
	cfg.SetExecErr(errDB)
	if _, err := s.Put(ctx, "app-1", "a.txt", "text/plain", []byte("x")); err != nil {
		t.Fatalf("Put with failing bookkeeping = %v, want success", err)
	}
	if ok, _ := bucket.Exists(ctx, "tenant-a/app-1/a.txt"); !ok {
		t.Error("the object was not stored")
	}
}

// SQL failures on the row-backed paths surface with the step they broke.
func TestAssets_SQLFailures(t *testing.T) {
	ctx := assetCtx("tenant-a")

	s, _, cfg := newFaultyAssetsTest(t)
	cfg.SetQueryErr(errDB)
	if _, err := s.ListFolder(ctx, "app-1", "images"); err == nil || !strings.Contains(err.Error(), `reading asset folder rows "images"`) {
		t.Errorf("ListFolder with failing rows = %v", err)
	}

	s, _, cfg = newFaultyAssetsTest(t)
	cfg.PushRowData(&mockdb.RowData{Columns: []string{"id"}, Rows: [][]any{{"app-1"}}}) // the app exists
	cfg.PushQueryErr(errDB)                                                             // marking the first folder row fails
	if _, err := s.CreateFolder(ctx, "app-1", "images/icons"); err == nil ||
		!strings.Contains(err.Error(), `creating asset folder "images/icons"`) || !strings.Contains(err.Error(), "marking folder row") {
		t.Errorf("CreateFolder with failing rows = %v", err)
	}
	cfg.SetExecErr(errDB)
	if err := s.Rebuild(ctx, "app-1"); err == nil || !strings.Contains(err.Error(), "rebuilding asset bookkeeping") {
		t.Errorf("Rebuild with failing rows = %v", err)
	}

	s, _, cfg = newFaultyAssetsTest(t)
	cfg.SetRowData(nil) // no app row: the app does not exist for the tenant
	if _, err := s.CreateFolder(ctx, "app-1", "images"); !errors.Is(err, ErrNotFound) {
		t.Errorf("CreateFolder for an absent app = %v, want ErrNotFound", err)
	}
}

// Each of the four reads behind Stats reports its own failure.
func TestAssets_StatsFailures(t *testing.T) {
	ctx := assetCtx("tenant-a")
	for i, want := range []string{"reading root folder row", "reading kind rows", "reading day rows", "reading rebuild stamp"} {
		s, _, cfg := newFaultyAssetsTest(t)
		for range i {
			cfg.PushRowData(nil)
		}
		cfg.PushQueryErr(errDB)
		if _, err := s.Stats(ctx, "app-1"); err == nil || !strings.Contains(err.Error(), want) || !errors.Is(err, errDB) {
			t.Errorf("Stats with read %d failing = %v, want %q", i+1, err, want)
		}
	}
}

// failNth arranges for the k-th (0-based) statement on one of the mock
// driver's queues to fail: the driver answers each single-row INSERT from the
// query queue and every other write from the exec queue, in statement order,
// so the k responses before it are queued as successes and the rest fall
// back to the defaults.
func failNth(cfg *mockdb.Config, queue string, k int, err error) {
	for range k {
		if queue == "query" {
			cfg.PushRowData(&mockdb.RowData{Columns: []string{"id"}, Rows: [][]any{{"row-1"}}})
		} else {
			cfg.PushExecErr(nil)
		}
	}
	if queue == "query" {
		cfg.PushQueryErr(err)
	} else {
		cfg.PushExecErr(err)
	}
}

// A bookkeeping apply names the statement that failed. A delete touches
// every kind of statement — ensure, adjust, prune — for folders, kinds, and
// days: two ancestor folders, two kinds, and two day rows, ensured one by one
// and adjusted in batches of two.
func TestAssetBooks_ApplyFailures(t *testing.T) {
	ctx := context.Background()
	prev := &models.Asset{Key: "images/a.png", Size: 40, LastModified: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}
	change := deleteChange("images/a.png", models.KindImage, prev)
	cases := []struct {
		queue string
		want  string
		k     int
	}{
		{"query", `ensuring folder row ""`, 0},
		{"query", `ensuring folder row "images"`, 1},
		{"exec", "adjusting folder rows", 0},
		{"exec", "pruning empty folder rows", 2},
		{"query", `ensuring kind row "image"`, 2},
		{"exec", "adjusting kind rows", 3},
		{"query", `ensuring day row "image"/2026-09-01`, 4},
		{"exec", "adjusting day rows", 5},
		{"exec", "pruning empty day rows", 7},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			s, _, cfg := newAssetsCapture(t)
			failNth(cfg, tc.queue, tc.k, errDB)
			err := s.books.apply(ctx, "tenant-a", "app-1", change)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !errors.Is(err, errDB) {
				t.Errorf("apply = %v, want %q wrapping the db error", err, tc.want)
			}
		})
	}
	s, _, cfg := newAssetsCapture(t)
	failNth(cfg, "exec", 8, errDB) // one past the last write: the failure is never reached
	if err := s.books.apply(ctx, "tenant-a", "app-1", change); err != nil {
		t.Errorf("apply with a failure queued past its writes = %v", err)
	}
}

// A rebuild names the statement that failed, from clearing the old rows to
// stamping the new ones: one asset folds to two folder rows, written one by
// one, and a batch each of kind and day rows.
func TestAssetBooks_RebuildFailures(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 16, 18, 0, 0, 0, time.UTC)
	assets := []*models.Asset{{Key: "images/a.png", ContentType: "image/png", Size: 1, LastModified: now}}
	cases := []struct {
		queue string
		want  string
		k     int
	}{
		{"exec", "clearing folder rows", 0},
		{"exec", "zeroing explicit folder rows", 1},
		{"query", `writing folder row ""`, 0},
		{"query", `writing folder row "images"`, 1},
		{"exec", "clearing kind rows", 2},
		{"exec", "writing kind rows", 3},
		{"exec", "clearing day rows", 4},
		{"exec", "writing day rows", 5},
		{"query", "stamping rebuild", 2},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			s, _, cfg := newAssetsCapture(t)
			failNth(cfg, tc.queue, tc.k, errDB)
			err := s.books.rebuild(ctx, "tenant-a", "app-1", foldAssets("tenant-a", "app-1", assets, now), now)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !errors.Is(err, errDB) {
				t.Errorf("rebuild = %v, want %q wrapping the db error", err, tc.want)
			}
		})
	}
}

// The cross-tenant rebuild stops at the first failure with the count so far,
// whether enumerating apps or rebuilding one, and reads the next page after
// the last id of a full one.
func TestStores_RebuildAssets_ErrorsAndPaging(t *testing.T) {
	ctx := context.Background()
	sum.Reset()
	sum.New()
	db, capture, cfg := mockdb.NewWithConfig()
	cfg.SetRowData(&mockdb.RowData{Columns: []string{"id"}, Rows: [][]any{{"row-1"}}})
	bucket := &faultyBucket{BucketProvider: testkit.NewBucketProvider()}
	st := New(db, astqlpg.New(), testkit.NewSearchProvider(), bucket)

	cfg.PushQueryErr(errDB)
	if n, err := st.RebuildAssets(ctx); err == nil || !strings.Contains(err.Error(), "enumerating apps") || n != 0 {
		t.Errorf("RebuildAssets with failing enumeration = %d, %v", n, err)
	}

	cfg.PushRowData(&mockdb.RowData{Columns: []string{"id", "tenant_id", "name"}, Rows: [][]any{{"app-1", "tenant-a", "one"}}})
	bucket.listErr = errBucket
	if n, err := st.RebuildAssets(ctx); err == nil || !strings.Contains(err.Error(), "rebuilding assets of app app-1") || n != 0 {
		t.Errorf("RebuildAssets with a failing app = %d, %v", n, err)
	}
	bucket.listErr = nil

	// A full first page: the walk asks for the apps after its last id. The
	// default row answers that read as one more app, a short page that ends
	// the walk.
	full := make([][]any, 0, reindexBatch)
	for i := range reindexBatch {
		full = append(full, []any{fmt.Sprintf("app-%03d", i), "tenant-a", "app"})
	}
	cfg.PushRowData(&mockdb.RowData{Columns: []string{"id", "tenant_id", "name"}, Rows: full})
	before := len(capture.Queries)
	n, err := st.RebuildAssets(ctx)
	if err != nil || n != reindexBatch+1 {
		t.Errorf("RebuildAssets over a full page = %d, %v; want %d", n, err, reindexBatch+1)
	}
	pages := queriesOn(&mockdb.Capture{Queries: capture.Queries[before:]}, "SELECT", "apps")
	if len(pages) != 2 || !hasArg(pages[0], zeroUUID) || !hasArg(pages[1], "app-099") {
		t.Errorf("app enumeration reads = %d, want 2 (from the zero id, then after app-099)", len(pages))
	}
}
