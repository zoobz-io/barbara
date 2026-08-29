package stores

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
)

// BackfillActor is the system identity stamped as created_by on releases the
// backfill cuts — there is no acting user when a one-shot tool runs.
const BackfillActor = "00000000-0000-0000-0000-000000000000"

// backfillAppName is the deterministic name of the app seeded per tenant, so
// two runs on two environments agree.
const backfillAppName = "default"

// backfillPageSize pages the tenant scan by keyset.
const backfillPageSize = 500

// Backfill002Result summarizes what a backfill run changed.
type Backfill002Result struct {
	Tenants     int // tenants seeded (skipped tenants not counted)
	Collections int // collections created
	Documents   int // documents placed in the tree
	Releases    int // release 1s cut (at most one per tenant)
}

// Backfill002 seeds the plan-002 tree from existing path-like keys: one app
// per tenant, a collection per key prefix, tree placement on every document —
// and release 1 per app, cut from the published_version_id pointers, so
// existing publish state survives the pointer's removal. Keys are unchanged.
//
// One transaction per tenant, and a tenant that already has an app is skipped,
// so the run is idempotent — rerunning adds no duplicate rows. No domain
// events are emitted and no projection jobs are enqueued: the backfill replays
// no domain activity, it re-shapes rows that already exist, and the OpenSearch
// index already matches the pointers release 1 snapshots.
//
// Tenant-agnostic operational machinery (cf. Reindex): it runs outside any
// tenant context and must see every tenant's documents. Not exposed on any
// surface — cmd/backfill002 is the only caller.
func (s *Stores) Backfill002(ctx context.Context) (*Backfill002Result, error) {
	apps := sum.NewDatabase[models.App](s.db, "apps", s.renderer)
	collections := sum.NewDatabase[models.Collection](s.db, "collections", s.renderer)
	releases := sum.NewDatabase[models.Release](s.db, "releases", s.renderer)
	entries := sum.NewDatabase[models.ReleaseEntry](s.db, "release_entries", s.renderer)

	tenants, err := s.backfillTenants(ctx)
	if err != nil {
		return nil, err
	}

	result := &Backfill002Result{}
	for _, tenantID := range tenants {
		n, err := apps.Count().
			Where("tenant_id", "=", "tenant_id").
			Exec(ctx, map[string]any{"tenant_id": tenantID})
		if err != nil {
			return nil, fmt.Errorf("checking apps for tenant %s: %w", tenantID, err)
		}
		if n > 0 {
			continue // already seeded (or the tenant made an app post-002)
		}
		if err := s.inTx(ctx, func(tx *sqlx.Tx) error {
			return s.backfillTenant(ctx, tx, tenantID, apps, collections, releases, entries, result)
		}); err != nil {
			return nil, fmt.Errorf("backfilling tenant %s: %w", tenantID, err)
		}
		result.Tenants++
	}
	return result, nil
}

// backfillTenants scans for tenants that still have unplaced documents
// (app_id IS NULL), keyset-paged so a large table stays linear.
func (s *Stores) backfillTenants(ctx context.Context) ([]string, error) {
	seen := map[string]bool{}
	var tenants []string
	afterID := "00000000-0000-0000-0000-000000000000"
	for {
		docs, err := s.Documents.Query().
			WhereNull("app_id").
			Where("id", ">", "after_id").
			OrderBy("id", "asc").
			Limit(backfillPageSize).
			Exec(ctx, map[string]any{"after_id": afterID})
		if err != nil {
			return nil, fmt.Errorf("scanning unplaced documents: %w", err)
		}
		if len(docs) == 0 {
			return tenants, nil
		}
		for _, d := range docs {
			if !seen[d.TenantID] {
				seen[d.TenantID] = true
				tenants = append(tenants, d.TenantID)
			}
		}
		afterID = docs[len(docs)-1].ID
	}
}

// backfillTenant seeds one tenant inside tx: the app, the collection tree
// derived from key prefixes, placement on every document, and release 1 from
// the published pointers (only when something is published — an app with no
// live pages keeps a NULL current_release_id, which serves identically).
func (s *Stores) backfillTenant(
	ctx context.Context, tx *sqlx.Tx, tenantID string,
	apps *sum.Database[models.App], collections *sum.Database[models.Collection],
	releases *sum.Database[models.Release], entries *sum.Database[models.ReleaseEntry],
	result *Backfill002Result,
) error {
	now := time.Now()
	app, err := apps.Insert().ExecTx(ctx, tx, &models.App{
		TenantID: tenantID, Name: backfillAppName, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("creating app: %w", err)
	}

	docs, err := s.Documents.Query().
		Where("tenant_id", "=", "tenant_id").
		WhereNull("app_id").
		OrderBy("key", "asc").
		Exec(ctx, map[string]any{"tenant_id": tenantID})
	if err != nil {
		return fmt.Errorf("listing documents: %w", err)
	}

	collectionIDs := map[string]string{} // materialized path -> collection id
	var published []*models.Document
	for _, doc := range docs {
		path, name := splitKey(doc.Key)
		var collectionID any // nil = app root
		if len(path) > 0 {
			id, collErr := ensureCollections(ctx, tx, collections, tenantID, app.ID, path, collectionIDs, result)
			if collErr != nil {
				return collErr
			}
			collectionID = id
		}
		if _, placeErr := s.Documents.Modify().
			Set("app_id", "app_id").
			Set("collection_id", "collection_id").
			Set("name", "name").
			Set("updated_at", "updated_at").
			Where("id", "=", "id").
			ExecTx(ctx, tx, map[string]any{
				"app_id":        app.ID,
				"collection_id": collectionID,
				"name":          name,
				"updated_at":    now,
				"id":            doc.ID,
			}); placeErr != nil {
			return fmt.Errorf("placing document %s: %w", doc.ID, placeErr)
		}
		result.Documents++
		if doc.PublishedVersionID != nil {
			published = append(published, doc)
		}
	}

	if len(published) == 0 {
		return nil
	}
	release, err := releases.Insert().ExecTx(ctx, tx, &models.Release{
		AppID: app.ID, TenantID: tenantID, Number: 1,
		CreatedBy: BackfillActor, CreatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("cutting release 1: %w", err)
	}
	for _, doc := range published {
		if _, err := entries.Insert().ExecTx(ctx, tx, &models.ReleaseEntry{
			ReleaseID: release.ID, Key: doc.Key,
			DocumentID: doc.ID, VersionID: *doc.PublishedVersionID,
		}); err != nil {
			return fmt.Errorf("writing release entry for %s: %w", doc.Key, err)
		}
	}
	if _, err := apps.Modify().
		Set("current_release_id", "current_release_id").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		ExecTx(ctx, tx, map[string]any{
			"current_release_id": release.ID,
			"updated_at":         now,
			"id":                 app.ID,
		}); err != nil {
		return fmt.Errorf("pointing app at release 1: %w", err)
	}
	result.Releases++
	return nil
}

// ensureCollections walks a collection path, creating each missing segment
// under its parent and caching ids by materialized path. Returns the id of the
// deepest collection.
func ensureCollections(
	ctx context.Context, tx *sqlx.Tx, collections *sum.Database[models.Collection],
	tenantID, appID string, path []string, cache map[string]string, result *Backfill002Result,
) (string, error) {
	var parentID *string
	var walked string
	for _, segment := range path {
		if walked == "" {
			walked = segment
		} else {
			walked = walked + "/" + segment
		}
		if id, ok := cache[walked]; ok {
			parentID = &id
			continue
		}
		now := time.Now()
		created, err := collections.Insert().ExecTx(ctx, tx, &models.Collection{
			TenantID: tenantID, AppID: appID, ParentID: parentID, Name: segment,
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return "", fmt.Errorf("creating collection %q: %w", walked, err)
		}
		cache[walked] = created.ID
		id := created.ID
		parentID = &id
		result.Collections++
	}
	return *parentID, nil
}

// splitKey derives tree placement from a path-like key: the last segment is
// the document name, the earlier segments the collection path. Empty segments
// (doubled or leading slashes) are dropped — slashes were convention, not
// hierarchy. A degenerate key with no usable segments (all slashes) lands at
// the app root under its verbatim key, which stays unique per tenant.
func splitKey(key string) (path []string, name string) {
	var segments []string
	for _, seg := range strings.Split(key, "/") {
		if seg != "" {
			segments = append(segments, seg)
		}
	}
	if len(segments) == 0 {
		return nil, key
	}
	return segments[:len(segments)-1], segments[len(segments)-1]
}
