package stores

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
)

// dailyWindow is the longest window the daily series serves; the rebuild
// prunes days older than this.
const dailyWindow = 90 * 24 * time.Hour

// assetBooks is the bookkeeping side of the assets store: the folder rollups,
// the per-kind stats, the daily series, and the rebuild stamp. Object storage
// is the source of truth; these rows are projections. Every write and delete
// adjusts exactly the rows the key touches (its ancestor folders, its kind,
// its day) inside one transaction, and Rebuild replaces an app's rows from a
// full listing when something drifted.
//
// Each adjustment is two builder statements per table — an upsert that makes
// the row exist and an increment on it — because the conflict clause assigns
// values rather than expressions. Both run in the same transaction, so the
// pair is atomic and race-free.
type assetBooks struct {
	db      *sqlx.DB
	folders *sum.Database[models.AssetFolderStat]
	kinds   *sum.Database[models.AssetKindStat]
	days    *sum.Database[models.AssetDayStat]
	marks   *sum.Database[models.AssetBookkeeping]
	now     func() time.Time
}

func newAssetBooks(db *sqlx.DB, renderer astql.Renderer) *assetBooks {
	return &assetBooks{
		db:      db,
		folders: sum.NewDatabase[models.AssetFolderStat](db, "asset_folders", renderer),
		kinds:   sum.NewDatabase[models.AssetKindStat](db, "asset_stats", renderer),
		days:    sum.NewDatabase[models.AssetDayStat](db, "asset_stats_daily", renderer),
		marks:   sum.NewDatabase[models.AssetBookkeeping](db, "asset_bookkeeping", renderer),
		now:     time.Now,
	}
}

// assetChange is one write or delete, reduced to what the bookkeeping needs:
// which rows to touch and by how much. A write to a new key adds one object;
// an overwrite adds the size difference and moves the object from its old day
// to today; a delete removes the object from every row it was counted in.
type assetChange struct {
	// PrevLastModified is when the previous object was written, when there
	// was one (an overwrite or a delete) and the bucket reported it.
	PrevLastModified time.Time
	Key              string
	Kind             models.Kind
	// Count is the object-count delta: +1 new, 0 overwrite, -1 delete.
	Count int64
	// Bytes is the size delta.
	Bytes int64
	// Size is the written object's size (writes only).
	Size int64
	// PrevSize is the previous object's size (PrevExisted only).
	PrevSize int64
	// Written is true for a write (the rows' last-written moves to now).
	Written bool
	// PrevExisted is true when the key held an object before the change.
	PrevExisted bool
}

// writeChange describes storing size bytes of kind at key over prev (nil for
// a new key).
func writeChange(key string, kind models.Kind, size int64, prev *models.Asset) assetChange {
	c := assetChange{Key: key, Kind: kind, Count: 1, Bytes: size, Written: true, Size: size}
	if prev != nil {
		c.Count = 0
		c.Bytes = size - prev.Size
		c.PrevExisted = true
		c.PrevSize = prev.Size
		c.PrevLastModified = prev.LastModified
	}
	return c
}

// deleteChange describes removing prev at key.
func deleteChange(key string, kind models.Kind, prev *models.Asset) assetChange {
	return assetChange{
		Key: key, Kind: kind, Count: -1, Bytes: -prev.Size,
		PrevExisted: true, PrevSize: prev.Size, PrevLastModified: prev.LastModified,
	}
}

// ancestorPaths lists the folder paths a key sits under, root first: "a/b/c.png"
// is under "", "a", and "a/b". Every key is under the root.
func ancestorPaths(key string) []string {
	paths := []string{""}
	for i, r := range key {
		if r == '/' {
			paths = append(paths, key[:i])
		}
	}
	return paths
}

// utcDay truncates t to its UTC calendar day.
func utcDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// apply adjusts the rows the changes touch, all in one transaction — a
// move is a delete at one key and a write at another, and the rows must
// never show the object in both places or neither.
func (b *assetBooks) apply(ctx context.Context, tenantID, appID string, changes ...assetChange) error {
	now := b.now()
	return inTx(ctx, b.db, func(tx *sqlx.Tx) error {
		for _, c := range changes {
			if err := b.applyFolders(ctx, tx, tenantID, appID, c, now); err != nil {
				return err
			}
			if err := b.applyKinds(ctx, tx, tenantID, appID, c, now); err != nil {
				return err
			}
			if err := b.applyDays(ctx, tx, tenantID, appID, c, now); err != nil {
				return err
			}
		}
		return nil
	})
}

// applyFolders ensures a row for every ancestor path, adds the deltas, and
// drops non-explicit rows that reached zero.
func (b *assetBooks) applyFolders(ctx context.Context, tx *sqlx.Tx, tenantID, appID string, c assetChange, now time.Time) error {
	paths := ancestorPaths(c.Key)
	for _, path := range paths {
		row := &models.AssetFolderStat{TenantID: tenantID, AppID: appID, Path: path, CreatedAt: now, UpdatedAt: now}
		if _, err := b.folders.Insert().
			OnConflict("tenant_id", "app_id", "path").
			DoUpdate().Set("updated_at", "updated_at").Build().
			ExecTx(ctx, tx, row); err != nil {
			return fmt.Errorf("ensuring folder row %q: %w", path, err)
		}
	}

	update := b.folders.Modify().
		SetExpr("count", "+", "d_count").
		SetExpr("bytes", "+", "d_bytes").
		Set("updated_at", "updated_at")
	if c.Written {
		update = update.Set("last_written_at", "last_written_at")
	}
	update = update.
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		Where("path", "=", "path")
	params := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		p := map[string]any{
			"d_count": c.Count, "d_bytes": c.Bytes, "updated_at": now,
			"tenant_id": tenantID, "app_id": appID, "path": path,
		}
		if c.Written {
			p["last_written_at"] = now
		}
		params = append(params, p)
	}
	if _, err := update.ExecBatchTx(ctx, tx, params); err != nil {
		return fmt.Errorf("adjusting folder rows: %w", err)
	}

	if c.Count < 0 {
		if _, err := b.folders.Remove().
			Where("tenant_id", "=", "tenant_id").
			Where("app_id", "=", "app_id").
			Where("count", "<=", "zero").
			Where("explicit", "=", "explicit").
			ExecTx(ctx, tx, map[string]any{"tenant_id": tenantID, "app_id": appID, "zero": 0, "explicit": false}); err != nil {
			return fmt.Errorf("pruning empty folder rows: %w", err)
		}
	}
	return nil
}

// applyKinds adjusts the key's kind row and the all-kinds row.
func (b *assetBooks) applyKinds(ctx context.Context, tx *sqlx.Tx, tenantID, appID string, c assetChange, now time.Time) error {
	kinds := []models.Kind{c.Kind, ""}
	for _, kind := range kinds {
		row := &models.AssetKindStat{TenantID: tenantID, AppID: appID, Kind: kind}
		if _, err := b.kinds.Insert().
			OnConflict("tenant_id", "app_id", "kind").
			DoUpdate().Set("kind", "kind").Build().
			ExecTx(ctx, tx, row); err != nil {
			return fmt.Errorf("ensuring kind row %q: %w", kind, err)
		}
	}

	update := b.kinds.Modify().
		SetExpr("count", "+", "d_count").
		SetExpr("bytes", "+", "d_bytes")
	if c.Written {
		update = update.Set("last_written_at", "last_written_at")
	}
	update = update.
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		Where("kind", "=", "kind")
	params := make([]map[string]any, 0, len(kinds))
	for _, kind := range kinds {
		p := map[string]any{
			"d_count": c.Count, "d_bytes": c.Bytes,
			"tenant_id": tenantID, "app_id": appID, "kind": string(kind),
		}
		if c.Written {
			p["last_written_at"] = now
		}
		params = append(params, p)
	}
	if _, err := update.ExecBatchTx(ctx, tx, params); err != nil {
		return fmt.Errorf("adjusting kind rows: %w", err)
	}
	return nil
}

// applyDays moves the object between day buckets: a write lands in today's
// bucket for its kind and for all kinds; an overwrite or delete also leaves
// the previous object's day, when the bucket reported one.
func (b *assetBooks) applyDays(ctx context.Context, tx *sqlx.Tx, tenantID, appID string, c assetChange, now time.Time) error {
	type dayDelta struct {
		day   time.Time
		kind  models.Kind
		count int64
		bytes int64
	}
	var deltas []dayDelta
	if c.Written {
		today := utcDay(now)
		deltas = append(deltas,
			dayDelta{today, c.Kind, 1, c.Size},
			dayDelta{today, "", 1, c.Size},
		)
	}
	if c.PrevExisted && !c.PrevLastModified.IsZero() {
		prevDay := utcDay(c.PrevLastModified)
		deltas = append(deltas,
			dayDelta{prevDay, c.Kind, -1, -c.PrevSize},
			dayDelta{prevDay, "", -1, -c.PrevSize},
		)
	}
	if len(deltas) == 0 {
		return nil
	}

	for _, d := range deltas {
		row := &models.AssetDayStat{TenantID: tenantID, AppID: appID, Kind: d.kind, Day: d.day}
		if _, err := b.days.Insert().
			OnConflict("tenant_id", "app_id", "kind", "day").
			DoUpdate().Set("day", "day").Build().
			ExecTx(ctx, tx, row); err != nil {
			return fmt.Errorf("ensuring day row %q/%s: %w", d.kind, d.day.Format(time.DateOnly), err)
		}
	}

	params := make([]map[string]any, 0, len(deltas))
	for _, d := range deltas {
		params = append(params, map[string]any{
			"d_count": d.count, "d_bytes": d.bytes,
			"tenant_id": tenantID, "app_id": appID, "kind": string(d.kind), "day": d.day,
		})
	}
	if _, err := b.days.Modify().
		SetExpr("count", "+", "d_count").
		SetExpr("bytes", "+", "d_bytes").
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		Where("kind", "=", "kind").
		Where("day", "=", "day").
		ExecBatchTx(ctx, tx, params); err != nil {
		return fmt.Errorf("adjusting day rows: %w", err)
	}

	if c.PrevExisted {
		if _, err := b.days.Remove().
			Where("tenant_id", "=", "tenant_id").
			Where("app_id", "=", "app_id").
			Where("count", "<=", "zero").
			ExecTx(ctx, tx, map[string]any{"tenant_id": tenantID, "app_id": appID, "zero": 0}); err != nil {
			return fmt.Errorf("pruning empty day rows: %w", err)
		}
	}
	return nil
}

// assetFold is a full listing folded into bookkeeping rows: what the tables
// should hold for the app right now.
type assetFold struct {
	Folders map[string]*models.AssetFolderStat
	Kinds   map[models.Kind]*models.AssetKindStat
	Days    map[string]*models.AssetDayStat // keyed "<kind>|<day>"
}

// foldAssets computes the bookkeeping rows for a listing as of now. Every key
// counts toward each ancestor path, its kind and the all-kinds row, and — when
// the bucket reported a last-modified within the daily window — its UTC day.
func foldAssets(tenantID, appID string, assets []*models.Asset, now time.Time) *assetFold {
	f := &assetFold{
		Folders: map[string]*models.AssetFolderStat{},
		Kinds:   map[models.Kind]*models.AssetKindStat{},
		Days:    map[string]*models.AssetDayStat{},
	}
	folder := func(path string) *models.AssetFolderStat {
		row, ok := f.Folders[path]
		if !ok {
			row = &models.AssetFolderStat{TenantID: tenantID, AppID: appID, Path: path, CreatedAt: now, UpdatedAt: now}
			f.Folders[path] = row
		}
		return row
	}
	kind := func(k models.Kind) *models.AssetKindStat {
		row, ok := f.Kinds[k]
		if !ok {
			row = &models.AssetKindStat{TenantID: tenantID, AppID: appID, Kind: k}
			f.Kinds[k] = row
		}
		return row
	}
	day := func(k models.Kind, d time.Time) *models.AssetDayStat {
		key := string(k) + "|" + d.Format(time.DateOnly)
		row, ok := f.Days[key]
		if !ok {
			row = &models.AssetDayStat{TenantID: tenantID, AppID: appID, Kind: k, Day: d}
			f.Days[key] = row
		}
		return row
	}
	newest := func(cur *time.Time, t time.Time) *time.Time {
		if t.IsZero() || (cur != nil && !t.After(*cur)) {
			return cur
		}
		return &t
	}

	folder("") // the root row exists even for an empty app
	kind("")
	cutoff := utcDay(now.Add(-dailyWindow))
	for _, a := range assets {
		k := models.KindOf(a.ContentType)
		for _, path := range ancestorPaths(a.Key) {
			row := folder(path)
			row.Count++
			row.Bytes += a.Size
			row.LastWrittenAt = newest(row.LastWrittenAt, a.LastModified)
		}
		for _, kk := range []models.Kind{k, ""} {
			row := kind(kk)
			row.Count++
			row.Bytes += a.Size
			row.LastWrittenAt = newest(row.LastWrittenAt, a.LastModified)
		}
		if a.LastModified.IsZero() {
			continue
		}
		d := utcDay(a.LastModified)
		if d.Before(cutoff) {
			continue
		}
		for _, kk := range []models.Kind{k, ""} {
			row := day(kk, d)
			row.Count++
			row.Bytes += a.Size
		}
	}
	return f
}

// rebuild replaces the app's bookkeeping with the fold, in one transaction:
// non-explicit folder rows not in the fold go away, explicit ones are zeroed
// then updated in place, and the kind and day tables are rewritten whole.
func (b *assetBooks) rebuild(ctx context.Context, tenantID, appID string, f *assetFold, now time.Time) error {
	scope := map[string]any{"tenant_id": tenantID, "app_id": appID}
	return inTx(ctx, b.db, func(tx *sqlx.Tx) error {
		// Folders: drop derived rows, zero explicit ones, then upsert the fold.
		// An explicit folder in the fold gets its real numbers; one outside it
		// stays at zero, which is exactly what an empty explicit folder is.
		if _, err := b.folders.Remove().
			Where("tenant_id", "=", "tenant_id").
			Where("app_id", "=", "app_id").
			Where("explicit", "=", "explicit").
			ExecTx(ctx, tx, map[string]any{"tenant_id": tenantID, "app_id": appID, "explicit": false}); err != nil {
			return fmt.Errorf("clearing folder rows: %w", err)
		}
		if _, err := b.folders.Modify().
			Set("count", "zero").
			Set("bytes", "zero").
			Set("last_written_at", "none").
			Set("updated_at", "updated_at").
			Where("tenant_id", "=", "tenant_id").
			Where("app_id", "=", "app_id").
			ExecBatchTx(ctx, tx, []map[string]any{{
				"zero": 0, "none": nil, "updated_at": now, "tenant_id": tenantID, "app_id": appID,
			}}); err != nil {
			return fmt.Errorf("zeroing explicit folder rows: %w", err)
		}
		for _, row := range f.Folders {
			if _, err := b.folders.Insert().
				OnConflict("tenant_id", "app_id", "path").
				DoUpdate().
				Set("count", "count").
				Set("bytes", "bytes").
				Set("last_written_at", "last_written_at").
				Set("updated_at", "updated_at").
				Build().
				ExecTx(ctx, tx, row); err != nil {
				return fmt.Errorf("writing folder row %q: %w", row.Path, err)
			}
		}

		// Kinds and days: rewrite whole.
		if _, err := b.kinds.Remove().
			Where("tenant_id", "=", "tenant_id").
			Where("app_id", "=", "app_id").
			ExecTx(ctx, tx, scope); err != nil {
			return fmt.Errorf("clearing kind rows: %w", err)
		}
		kinds := make([]*models.AssetKindStat, 0, len(f.Kinds))
		for _, row := range f.Kinds {
			kinds = append(kinds, row)
		}
		if _, err := b.kinds.Insert().ExecBatchTx(ctx, tx, kinds); err != nil {
			return fmt.Errorf("writing kind rows: %w", err)
		}
		if _, err := b.days.Remove().
			Where("tenant_id", "=", "tenant_id").
			Where("app_id", "=", "app_id").
			ExecTx(ctx, tx, scope); err != nil {
			return fmt.Errorf("clearing day rows: %w", err)
		}
		days := make([]*models.AssetDayStat, 0, len(f.Days))
		for _, row := range f.Days {
			days = append(days, row)
		}
		if _, err := b.days.Insert().ExecBatchTx(ctx, tx, days); err != nil {
			return fmt.Errorf("writing day rows: %w", err)
		}

		mark := &models.AssetBookkeeping{TenantID: tenantID, AppID: appID, ComputedAt: now}
		if _, err := b.marks.Insert().
			OnConflict("tenant_id", "app_id").
			DoUpdate().Set("computed_at", "computed_at").Build().
			ExecTx(ctx, tx, mark); err != nil {
			return fmt.Errorf("stamping rebuild: %w", err)
		}
		return nil
	})
}

// children returns the folder rows directly beneath folderPath ("" for the
// root): those whose path has exactly one more segment. Postgres narrows to
// the subtree by prefix; the depth check is done here, since a LIKE cannot
// count segments. Rows beneath the root are every row but the root's own.
func (b *assetBooks) children(ctx context.Context, tenantID, appID, folderPath string) ([]*models.AssetFolderStat, error) {
	prefix := ""
	if folderPath != "" {
		prefix = folderPath + "/"
	}
	q := b.folders.Query().
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		OrderBy("path", "asc")
	params := map[string]any{"tenant_id": tenantID, "app_id": appID}
	if prefix != "" {
		q = q.Where("path", "LIKE", "pattern")
		params["pattern"] = escapeLike(prefix) + "%"
	}
	rows, err := q.Exec(ctx, params)
	if err != nil {
		return nil, err
	}
	direct := rows[:0]
	for _, row := range rows {
		rest := strings.TrimPrefix(row.Path, prefix)
		if rest == "" || strings.Contains(rest, "/") || !strings.HasPrefix(row.Path, prefix) {
			continue
		}
		direct = append(direct, row)
	}
	return direct, nil
}

// markExplicit makes folderPath and each of its ancestors (the root aside) an
// explicit folder row, creating any that are missing at a zero rollup. An
// existing row keeps its numbers and only gains the flag. One transaction.
func (b *assetBooks) markExplicit(ctx context.Context, tenantID, appID, folderPath string) error {
	now := b.now()
	return inTx(ctx, b.db, func(tx *sqlx.Tx) error {
		for _, p := range ancestorPaths(folderPath + "/") {
			if p == "" {
				continue
			}
			row := &models.AssetFolderStat{
				TenantID: tenantID, AppID: appID, Path: p, Explicit: true, CreatedAt: now, UpdatedAt: now,
			}
			if _, err := b.folders.Insert().
				OnConflict("tenant_id", "app_id", "path").
				DoUpdate().Set("explicit", "explicit").Set("updated_at", "updated_at").Build().
				ExecTx(ctx, tx, row); err != nil {
				return fmt.Errorf("marking folder row %q explicit: %w", p, err)
			}
		}
		return nil
	})
}

// stats reads the app-level view.
func (b *assetBooks) stats(ctx context.Context, tenantID, appID string) (*models.AssetStats, error) {
	scope := map[string]any{"tenant_id": tenantID, "app_id": appID}
	out := &models.AssetStats{Root: &models.AssetFolderStat{TenantID: tenantID, AppID: appID}}

	roots, err := b.folders.Query().
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		Where("path", "=", "path").
		Exec(ctx, map[string]any{"tenant_id": tenantID, "app_id": appID, "path": ""})
	if err != nil {
		return nil, fmt.Errorf("reading root folder row: %w", err)
	}
	if len(roots) > 0 {
		out.Root = roots[0]
	}

	kinds, err := b.kinds.Query().
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		Where("kind", "!=", "all").
		OrderBy("kind", "asc").
		Exec(ctx, map[string]any{"tenant_id": tenantID, "app_id": appID, "all": ""})
	if err != nil {
		return nil, fmt.Errorf("reading kind rows: %w", err)
	}
	out.Kinds = kinds

	days, err := b.days.Query().
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		OrderBy("day", "asc").
		Exec(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("reading day rows: %w", err)
	}
	out.Days = days

	marks, err := b.marks.Query().
		Where("tenant_id", "=", "tenant_id").
		Where("app_id", "=", "app_id").
		Exec(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("reading rebuild stamp: %w", err)
	}
	if len(marks) > 0 {
		out.ComputedAt = &marks[0].ComputedAt
	}
	return out, nil
}

// inTx runs fn in a transaction on db, committing on success and rolling back
// on error.
func inTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}
	return nil
}
