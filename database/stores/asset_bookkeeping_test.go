//go:build testing

package stores

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub/mockdb"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/testing/testkit"
)

func TestAncestorPaths(t *testing.T) {
	cases := map[string][]string{
		"README.md":             {""},
		"images/logo.png":       {"", "images"},
		"images/icons/menu.svg": {"", "images", "images/icons"},
		"a/b/c/d":               {"", "a", "a/b", "a/b/c"},
	}
	for key, want := range cases {
		if got := ancestorPaths(key); !reflect.DeepEqual(got, want) {
			t.Errorf("ancestorPaths(%q) = %v, want %v", key, got, want)
		}
	}
}

// A new key is one more object; an overwrite is a size change that leaves its
// old day; a delete removes what was there.
func TestAssetChange(t *testing.T) {
	old := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	prev := &models.Asset{Key: "a.png", Size: 40, LastModified: old}

	if c := writeChange("a.png", models.KindImage, 100, nil); c.Count != 1 || c.Bytes != 100 || !c.Written || c.PrevExisted {
		t.Errorf("new write = %+v", c)
	}
	if c := writeChange("a.png", models.KindImage, 100, prev); c.Count != 0 || c.Bytes != 60 || !c.Written ||
		!c.PrevExisted || c.PrevSize != 40 || !c.PrevLastModified.Equal(old) {
		t.Errorf("overwrite = %+v", c)
	}
	if c := deleteChange("a.png", models.KindImage, prev); c.Count != -1 || c.Bytes != -40 || c.Written ||
		!c.PrevExisted || c.PrevSize != 40 || !c.PrevLastModified.Equal(old) {
		t.Errorf("delete = %+v", c)
	}
}

// The fold is what the tables should hold: every key counted into each
// ancestor path, its kind and the all-kinds row, and its UTC day when the
// bucket reported one within the window. Newest write wins for last-written.
func TestFoldAssets(t *testing.T) {
	now := time.Date(2026, 9, 16, 18, 0, 0, 0, time.UTC)
	d := func(daysAgo int) time.Time { return now.Add(-time.Duration(daysAgo) * 24 * time.Hour) }
	assets := []*models.Asset{
		{Key: "README.md", ContentType: "text/markdown", Size: 1, LastModified: d(0)},
		{Key: "images/logo.png", ContentType: "image/png", Size: 10, LastModified: d(1)},
		{Key: "images/icons/menu.svg", ContentType: "image/svg+xml", Size: 100, LastModified: d(1)},
		{Key: "docs/spec.pdf", ContentType: "application/pdf", Size: 1000, LastModified: d(200)}, // outside the window
		{Key: "docs/old.txt", ContentType: "text/plain", Size: 10000},                            // no last-modified
	}

	f := foldAssets("t", "app", assets, now)

	folders := map[string][2]int64{}
	for path, row := range f.Folders {
		folders[path] = [2]int64{row.Count, row.Bytes}
	}
	wantFolders := map[string][2]int64{
		"":             {5, 11111},
		"images":       {2, 110},
		"images/icons": {1, 100},
		"docs":         {2, 11000},
	}
	if !reflect.DeepEqual(folders, wantFolders) {
		t.Errorf("folders = %v, want %v", folders, wantFolders)
	}
	if lw := f.Folders[""].LastWrittenAt; lw == nil || !lw.Equal(d(0)) {
		t.Errorf("root last written = %v, want %v", lw, d(0))
	}
	if lw := f.Folders["docs"].LastWrittenAt; lw == nil || !lw.Equal(d(200)) {
		t.Errorf("docs last written = %v, want %v (zero times never win)", lw, d(200))
	}

	kinds := map[models.Kind][2]int64{}
	for k, row := range f.Kinds {
		kinds[k] = [2]int64{row.Count, row.Bytes}
	}
	wantKinds := map[models.Kind][2]int64{
		"":               {5, 11111},
		models.KindText:  {3, 11001},
		models.KindImage: {2, 110},
	}
	if !reflect.DeepEqual(kinds, wantKinds) {
		t.Errorf("kinds = %v, want %v", kinds, wantKinds)
	}

	days := map[string][2]int64{}
	for key, row := range f.Days {
		days[key] = [2]int64{row.Count, row.Bytes}
	}
	today, yesterday := utcDay(d(0)).Format(time.DateOnly), utcDay(d(1)).Format(time.DateOnly)
	wantDays := map[string][2]int64{
		"text|" + today:      {1, 1},
		"|" + today:          {1, 1},
		"image|" + yesterday: {2, 110},
		"|" + yesterday:      {2, 110},
	}
	if !reflect.DeepEqual(days, wantDays) {
		t.Errorf("days = %v, want %v", days, wantDays)
	}
}

// An empty app still folds to a zero root row and a zero all-kinds row, so
// the stats read never has to invent them.
func TestFoldAssets_Empty(t *testing.T) {
	f := foldAssets("t", "app", nil, time.Now())
	if len(f.Folders) != 1 || f.Folders[""] == nil || f.Folders[""].Count != 0 {
		t.Errorf("folders = %v, want just a zero root", f.Folders)
	}
	if len(f.Kinds) != 1 || f.Kinds[""] == nil {
		t.Errorf("kinds = %v, want just a zero all-kinds row", f.Kinds)
	}
	if len(f.Days) != 0 {
		t.Errorf("days = %v, want none", f.Days)
	}
}

// newAssetsCapture is newAssetsTest with the SQL capture, and a fallback row
// so every ensure-upsert (INSERT ... RETURNING) hands back a row as Postgres
// would. Queue an appRow before each Put for the app guard.
func newAssetsCapture(t *testing.T) (*Assets, *mockdb.Capture, *mockdb.Config) {
	t.Helper()
	sum.Reset()
	sum.New()
	db, capture, cfg := mockdb.NewWithConfig()
	cfg.SetRowData(&mockdb.RowData{Columns: []string{"id"}, Rows: [][]any{{"row-1"}}})
	s := NewAssets(testkit.NewBucketProvider(), NewApps(db, astqlpg.New()), db, astqlpg.New())
	s.books.now = func() time.Time { return time.Date(2026, 9, 16, 18, 0, 0, 0, time.UTC) }
	return s, capture, cfg
}

// queriesOn returns the captured statements against table that start with verb.
func queriesOn(capture *mockdb.Capture, verb, table string) []mockdb.CapturedQuery {
	var out []mockdb.CapturedQuery
	for _, q := range capture.Queries {
		if strings.HasPrefix(q.Query, verb) && strings.Contains(q.Query, `"`+table+`"`) {
			out = append(out, q)
		}
	}
	return out
}

func hasArg(q mockdb.CapturedQuery, want any) bool {
	for _, a := range q.Args {
		if reflect.DeepEqual(a, want) {
			return true
		}
	}
	return false
}

// A write to a nested key touches exactly its ancestors — root, images,
// images/icons — plus its kind and the all-kinds row, plus today's day
// bucket for both, and nothing else. Each row is ensured then incremented.
func TestAssets_Put_Bookkeeping(t *testing.T) {
	s, capture, cfg := newAssetsCapture(t)
	ctx := assetCtx("tenant-a")

	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "images/icons/menu.svg", "image/svg+xml", []byte("<svg/>")); err != nil {
		t.Fatalf("put: %v", err)
	}

	ensured := queriesOn(capture, "INSERT", "asset_folders")
	if len(ensured) != 3 {
		t.Fatalf("folder rows ensured = %d, want 3 (root, images, images/icons)", len(ensured))
	}
	for i, path := range []string{"", "images", "images/icons"} {
		if !hasArg(ensured[i], path) {
			t.Errorf("folder ensure %d args = %v, want path %q", i, ensured[i].Args, path)
		}
	}
	bumped := queriesOn(capture, "UPDATE", "asset_folders")
	if len(bumped) != 3 {
		t.Fatalf("folder rows incremented = %d, want 3", len(bumped))
	}
	for _, q := range bumped {
		assignments, _, _ := strings.Cut(q.Query, " WHERE ")
		if !strings.Contains(assignments, `"count" + `) || !strings.Contains(assignments, `"bytes" + `) ||
			!strings.Contains(assignments, `"last_written_at" = `) {
			t.Errorf("folder increment = %s, want count/bytes increments and last_written_at", q.Query)
		}
		if !hasArg(q, int64(1)) || !hasArg(q, int64(6)) {
			t.Errorf("folder increment args = %v, want +1 object and +6 bytes", q.Args)
		}
	}
	if pruned := queriesOn(capture, "DELETE", "asset_folders"); len(pruned) != 0 {
		t.Errorf("a write pruned folder rows: %d deletes", len(pruned))
	}

	kinds := queriesOn(capture, "INSERT", "asset_stats")
	if len(kinds) != 2 || !hasArg(kinds[0], "image") || !hasArg(kinds[1], "") {
		t.Errorf("kind rows ensured = %v, want image then all-kinds", kinds)
	}
	if n := len(queriesOn(capture, "UPDATE", "asset_stats")); n != 2 {
		t.Errorf("kind rows incremented = %d, want 2", n)
	}

	today := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	days := queriesOn(capture, "INSERT", "asset_stats_daily")
	if len(days) != 2 || !hasArg(days[0], today) || !hasArg(days[1], today) {
		t.Errorf("day rows ensured = %v, want two for today", days)
	}
	if n := len(queriesOn(capture, "UPDATE", "asset_stats_daily")); n != 2 {
		t.Errorf("day rows incremented = %d, want 2", n)
	}
	if n := len(queriesOn(capture, "DELETE", "asset_stats_daily")); n != 0 {
		t.Errorf("a new write pruned day rows: %d deletes", n)
	}
}

// Deleting adjusts the same rows downward, leaves last-written alone, and
// prunes folder and day rows that reached zero. The deleted object's day comes
// from the bucket's last-modified, so it leaves that bucket, not today's.
func TestAssets_Delete_Bookkeeping(t *testing.T) {
	s, capture, cfg := newAssetsCapture(t)
	ctx := assetCtx("tenant-a")
	written := time.Date(2026, 9, 10, 9, 0, 0, 0, time.UTC)
	fake, ok := s.bucket.(*testkit.BucketProvider)
	if !ok {
		t.Fatal("test store is not over the fake bucket")
	}
	fake.Now = func() time.Time { return written }

	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "docs/spec.pdf", "application/pdf", []byte("%PDF")); err != nil {
		t.Fatalf("put: %v", err)
	}
	capture.Reset()

	if err := s.Delete(ctx, "app-1", "docs/spec.pdf"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	bumped := queriesOn(capture, "UPDATE", "asset_folders")
	if len(bumped) != 2 {
		t.Fatalf("folder rows adjusted = %d, want 2 (root, docs)", len(bumped))
	}
	for _, q := range bumped {
		assignments, _, _ := strings.Cut(q.Query, " WHERE ")
		if strings.Contains(assignments, `"last_written_at"`) {
			t.Errorf("a delete moved last_written_at: %s", q.Query)
		}
		if !hasArg(q, int64(-1)) || !hasArg(q, int64(-4)) {
			t.Errorf("folder decrement args = %v, want -1 object and -4 bytes", q.Args)
		}
	}
	if n := len(queriesOn(capture, "DELETE", "asset_folders")); n != 1 {
		t.Errorf("empty folder prune = %d deletes, want 1", n)
	}
	days := queriesOn(capture, "INSERT", "asset_stats_daily")
	if len(days) != 2 || !hasArg(days[0], utcDay(written)) {
		t.Errorf("day rows touched = %v, want the written day, not today", days)
	}
	if n := len(queriesOn(capture, "DELETE", "asset_stats_daily")); n != 1 {
		t.Errorf("empty day prune = %d deletes, want 1", n)
	}
}

// Deleting a missing key is ErrNotFound from the pre-delete Stat: nothing
// reaches the bookkeeping.
func TestAssets_Delete_Missing_NoBookkeeping(t *testing.T) {
	s, capture, _ := newAssetsCapture(t)
	if err := s.Delete(assetCtx("tenant-a"), "app-1", "nothing.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing = %v, want ErrNotFound", err)
	}
	if len(capture.Queries) != 0 {
		t.Errorf("a missing delete issued %d statements, want none", len(capture.Queries))
	}
}

// Rebuild lists the app and replaces its rows in one transaction: derived
// folder rows cleared, explicit ones zeroed, every folded row upserted, kind
// and day tables rewritten, and the stamp set.
func TestAssets_Rebuild(t *testing.T) {
	s, capture, cfg := newAssetsCapture(t)
	ctx := assetCtx("tenant-a")
	for _, key := range []string{"README.md", "images/logo.png"} {
		cfg.PushRowData(appRow())
		if _, err := s.Put(ctx, "app-1", key, "", []byte("x")); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}
	capture.Reset()

	if err := s.Rebuild(ctx, "app-1"); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	if n := len(queriesOn(capture, "DELETE", "asset_folders")); n != 1 {
		t.Errorf("derived folder clear = %d deletes, want 1", n)
	}
	if n := len(queriesOn(capture, "UPDATE", "asset_folders")); n != 1 {
		t.Errorf("explicit folder zeroing = %d updates, want 1", n)
	}
	if n := len(queriesOn(capture, "INSERT", "asset_folders")); n != 2 {
		t.Errorf("folder rows written = %d, want 2 (root, images)", n)
	}
	if n := len(queriesOn(capture, "INSERT", "asset_stats")); n != 1 {
		t.Errorf("kind rows written in %d statements, want 1 batch", n)
	}
	if n := len(queriesOn(capture, "INSERT", "asset_bookkeeping")); n != 1 {
		t.Errorf("rebuild stamp = %d inserts, want 1", n)
	}
}

// children narrows to the subtree in SQL and to one depth here: beneath
// "images", images/icons is a child but images/icons/small is not, and the
// folder's own row is not its child. The root asks for every row, no LIKE.
func TestAssetBooks_Children(t *testing.T) {
	s, capture, cfg := newAssetsCapture(t)
	ctx := assetCtx("tenant-a")

	cfg.PushRowData(&mockdb.RowData{
		Columns: []string{"path", "count", "bytes", "explicit"},
		Rows: [][]any{
			{"images", int64(5), int64(50), false},
			{"images/icons", int64(2), int64(20), false},
			{"images/icons/small", int64(1), int64(10), false},
			{"images/shots", int64(0), int64(0), true},
		},
	})
	rows, err := s.books.children(ctx, "tenant-a", "app-1", "images")
	if err != nil {
		t.Fatalf("children: %v", err)
	}
	var got []string
	for _, r := range rows {
		got = append(got, r.Path)
	}
	if want := []string{"images/icons", "images/shots"}; !reflect.DeepEqual(got, want) {
		t.Errorf("children(images) = %v, want %v", got, want)
	}
	q, _ := capture.Last()
	if !strings.Contains(q.Query, "LIKE") || !hasArg(q, `images/%`) {
		t.Errorf("subtree query = %s %v, want a LIKE on images/%%", q.Query, q.Args)
	}

	capture.Reset()
	cfg.PushRowData(&mockdb.RowData{
		Columns: []string{"path", "count", "bytes", "explicit"},
		Rows:    [][]any{{"", int64(5), int64(50), false}, {"images", int64(5), int64(50), false}, {"images/icons", int64(2), int64(20), false}},
	})
	rows, err = s.books.children(ctx, "tenant-a", "app-1", "")
	if err != nil {
		t.Fatalf("children root: %v", err)
	}
	if len(rows) != 1 || rows[0].Path != "images" {
		t.Errorf("children(root) = %+v, want images only", rows)
	}
	if q, _ := capture.Last(); strings.Contains(q.Query, "LIKE") {
		t.Errorf("root query narrows by LIKE: %s", q.Query)
	}
}

// Marking a nested folder explicit upserts it and each ancestor but the
// root, every one flagged explicit, in one transaction.
func TestAssetBooks_MarkExplicit(t *testing.T) {
	s, capture, _ := newAssetsCapture(t)
	if err := s.books.markExplicit(assetCtx("tenant-a"), "tenant-a", "app-1", "images/icons/small"); err != nil {
		t.Fatalf("markExplicit: %v", err)
	}
	upserts := queriesOn(capture, "INSERT", "asset_folders")
	if len(upserts) != 3 {
		t.Fatalf("folder upserts = %d, want 3 (images, images/icons, images/icons/small)", len(upserts))
	}
	for i, path := range []string{"images", "images/icons", "images/icons/small"} {
		if !hasArg(upserts[i], path) || !hasArg(upserts[i], true) {
			t.Errorf("upsert %d args = %v, want path %q flagged explicit", i, upserts[i].Args, path)
		}
		if !strings.Contains(upserts[i].Query, "ON CONFLICT") || !strings.Contains(upserts[i].Query, `"explicit"`) {
			t.Errorf("upsert %d does not set explicit on conflict: %s", i, upserts[i].Query)
		}
	}
	for _, q := range upserts {
		if hasArg(q, "") {
			t.Errorf("the root row was marked explicit: %v", q.Args)
		}
	}
}

// A move books the delete at the old key and the write at the new one in a
// single transaction: the old ancestors lose the object, the new ones gain
// it, and the kind rows net to zero.
func TestAssets_Move_Bookkeeping(t *testing.T) {
	s, capture, cfg := newAssetsCapture(t)
	ctx := assetCtx("tenant-a")
	cfg.PushRowData(appRow())
	if _, err := s.Put(ctx, "app-1", "images/logo.png", "image/png", []byte("png")); err != nil {
		t.Fatalf("put: %v", err)
	}
	capture.Reset()

	if _, err := s.Move(ctx, "app-1", "images/logo.png", "brand/logo.png"); err != nil {
		t.Fatalf("move: %v", err)
	}
	var begins, commits int
	for _, q := range capture.Queries {
		switch {
		case strings.HasPrefix(q.Query, "BEGIN"):
			begins++
		case strings.HasPrefix(q.Query, "COMMIT"):
			commits++
		}
	}
	if begins > 1 || commits > 1 {
		t.Errorf("move used %d transactions, want one", begins)
	}
	folders := queriesOn(capture, "UPDATE", "asset_folders")
	var minus, plus []string
	for _, q := range folders {
		for _, a := range q.Args {
			if p, ok := a.(string); ok && (p == "" || p == "images" || p == "brand") {
				if hasArg(q, int64(-1)) {
					minus = append(minus, p)
				} else if hasArg(q, int64(1)) {
					plus = append(plus, p)
				}
			}
		}
	}
	if !reflect.DeepEqual(minus, []string{"", "images"}) || !reflect.DeepEqual(plus, []string{"", "brand"}) {
		t.Errorf("folder deltas: -1 on %v, +1 on %v; want -1 on root+images and +1 on root+brand", minus, plus)
	}
}
