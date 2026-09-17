package stores

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"path"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// defaultAssetContentType is used when an upload declares no content type.
const defaultAssetContentType = "application/octet-stream"

// Extensions common to site assets that Go's builtin mime table lacks (and
// that a container without /etc/mime.types never learns). Listings infer
// types from extensions, so without these a README.md lists as octet-stream.
func init() {
	for ext, typ := range map[string]string{
		".md":       "text/markdown",
		".markdown": "text/markdown",
		".txt":      "text/plain",
		".csv":      "text/csv",
		".yaml":     "application/yaml",
		".yml":      "application/yaml",
		".ico":      "image/x-icon",
		".woff":     "font/woff",
		".woff2":    "font/woff2",
		".ttf":      "font/ttf",
		".otf":      "font/otf",
	} {
		_ = mime.AddExtensionType(ext, typ) // only errors without a leading dot
	}
}

// contentTypeForKey infers a media type from a key's extension. Bucket
// listings carry no per-object content type (an S3 listing omits it, and a
// Stat per object would be N+1), so List falls back to the extension. The
// stored type still governs downloads; this is listing metadata only. Any
// parameters (charset) are stripped to the bare media type.
func contentTypeForKey(key string) string {
	ct := mime.TypeByExtension(path.Ext(key))
	if ct == "" {
		return defaultAssetContentType
	}
	if mediaType, _, err := mime.ParseMediaType(ct); err == nil {
		return mediaType
	}
	return ct
}

// Assets is the data-access layer for binary assets in object storage. An
// asset is an opaque blob addressed by a key that is unique per app; there is
// no Postgres row and no versioning — putting the same key overwrites. Assets
// live outside the collection tree and outside releases: a folder is a key
// prefix by convention, and the live object is the only version there is.
//
// The backing bucket is shared, so every operation namespaces the stored
// object key with the tenant and app ids: one tenant's keys are invisible to
// another, and a listing only ever sees its own app's prefix. The apps store
// guards Put — writes create namespaces, so the app must exist for the tenant;
// reads and deletes need no guard, a wrong app is just a miss.
//
// Alongside the bucket, the store keeps bookkeeping rows in Postgres (folder
// rollups, per-kind stats, a daily series): projections adjusted by every
// write and delete and rebuilt from a listing by Rebuild. They are never the
// truth — a bookkeeping failure after a successful bucket write is reported
// as an event, not as a failed request.
type Assets struct {
	bucket grub.BucketProvider
	apps   *Apps
	books  *assetBooks
}

// NewAssets creates an assets store over the shared object-storage bucket,
// with its bookkeeping tables on the shared Postgres connection.
func NewAssets(bucket grub.BucketProvider, apps *Apps, db *sqlx.DB, renderer astql.Renderer) *Assets {
	return &Assets{bucket: bucket, apps: apps, books: newAssetBooks(db, renderer)}
}

// objectKey namespaces a user-supplied key under the request's tenant and app.
// The stored object name is "<tenant>/<app>/<key>"; List filters by that
// prefix, so a key can never address another tenant's or app's object.
func (s *Assets) objectKey(tenantID, appID, key string) string {
	return tenantID + "/" + appID + "/" + key
}

// Put stores data at key for the app, overwriting any existing asset with that
// key. The app must exist for the request's tenant (ErrNotFound otherwise) —
// writes create namespaces, and namespaces belong to real apps. An empty
// contentType defaults to octet-stream. It returns the stored asset's metadata
// (bytes omitted) — the normalized content type and size the reader will see.
func (s *Assets) Put(ctx context.Context, appID, key, contentType string, data []byte) (*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.apps.Get(ctx, appID); err != nil {
		return nil, err // ErrNotFound when the app is absent for the tenant
	}
	if contentType == "" {
		contentType = defaultAssetContentType
	}
	obj := s.objectKey(tenantID, appID, key)
	// The previous object, if any: an overwrite must be counted as a size
	// change, not a second asset, and must leave its old day bucket.
	prev, statErr := s.stat(ctx, obj, key)
	if statErr != nil {
		return nil, statErr
	}
	size := int64(len(data))
	info := &grub.ObjectInfo{Key: obj, ContentType: contentType, Size: size}
	if err := s.bucket.Put(ctx, obj, data, info); err != nil {
		return nil, fmt.Errorf("putting asset %q: %w", key, err)
	}
	s.book(ctx, tenantID, appID, key, writeChange(key, models.KindOf(contentType), size, prev))
	ev := events.AssetWrittenEvent{Key: key, TenantID: tenantID, AppID: appID, ContentType: contentType, Size: size}
	if prev != nil {
		ev.Overwrote, ev.PrevSize, ev.PrevLastModified = true, prev.Size, prev.LastModified
	}
	events.Asset.Written.Emit(ctx, ev)
	return &models.Asset{Key: key, ContentType: contentType, Size: size, LastModified: s.books.now()}, nil
}

// stat returns the app-facing metadata of the object at obj, or nil when there
// is none. The bucket may omit a listing-time content type; Stat carries the
// stored one, falling back to the key's extension.
func (s *Assets) stat(ctx context.Context, obj, key string) (*models.Asset, error) {
	info, err := s.bucket.Stat(ctx, obj)
	if err != nil {
		if errors.Is(err, grub.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("checking asset %q: %w", key, err)
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = contentTypeForKey(key)
	}
	return &models.Asset{Key: key, ContentType: contentType, Size: info.Size, LastModified: info.LastModified}, nil
}

// book applies a change to the bookkeeping rows. The bucket write already
// succeeded, so a failure here is reported, not returned: the object is the
// truth and the next Rebuild repairs the rows.
func (s *Assets) book(ctx context.Context, tenantID, appID, key string, changes ...assetChange) {
	if err := s.books.apply(ctx, tenantID, appID, changes...); err != nil {
		events.Asset.BookkeepingFailed.Emit(ctx, events.AssetBookkeepingFailedEvent{
			Key: key, TenantID: tenantID, AppID: appID, Err: err,
		})
	}
}

// Rebuild replaces the app's bookkeeping from a full listing of its objects:
// the repair and backfill path, run on demand or on a slow heartbeat, never
// by a write. One recursive listing, one transaction.
func (s *Assets) Rebuild(ctx context.Context, appID string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	return s.rebuildApp(ctx, tenantID, appID)
}

// rebuildApp is Rebuild for a known tenant: the cross-tenant rebuild command
// walks every app and has no request principal to carry one.
func (s *Assets) rebuildApp(ctx context.Context, tenantID, appID string) error {
	assets, err := s.listAll(ctx, tenantID, appID, "")
	if err != nil {
		return err
	}
	now := s.books.now()
	if err := s.books.rebuild(ctx, tenantID, appID, foldAssets(tenantID, appID, assets, now), now); err != nil {
		return fmt.Errorf("rebuilding asset bookkeeping: %w", err)
	}
	return nil
}

// Stats returns the app's bookkeeping view: root rollup, per-kind breakdown,
// daily series, and when it was last rebuilt.
func (s *Assets) Stats(ctx context.Context, appID string) (*models.AssetStats, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	return s.books.stats(ctx, tenantID, appID)
}

// Get returns the app's asset stored at key, bytes included. ErrNotFound when
// the app has no asset with that key.
func (s *Assets) Get(ctx context.Context, appID, key string) (*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	data, info, err := s.bucket.Get(ctx, s.objectKey(tenantID, appID, key))
	if err != nil {
		if errors.Is(err, grub.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting asset %q: %w", key, err)
	}
	return &models.Asset{Key: key, ContentType: info.ContentType, Size: info.Size, Data: data, LastModified: info.LastModified}, nil
}

// List returns metadata for the app's assets, without the bytes, scoped by
// object-key prefix. A non-empty keyPrefix narrows the listing to keys under
// it — the folder view, since an asset folder is a key prefix by convention.
func (s *Assets) List(ctx context.Context, appID, keyPrefix string) ([]*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	return s.listAll(ctx, tenantID, appID, keyPrefix)
}

// listAll is List for a known tenant: every object under the app's prefix,
// recursively, metadata only.
func (s *Assets) listAll(ctx context.Context, tenantID, appID, keyPrefix string) ([]*models.Asset, error) {
	scope := s.objectKey(tenantID, appID, "")
	infos, err := s.bucket.List(ctx, scope+keyPrefix, 0)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}
	assets := make([]*models.Asset, 0, len(infos))
	for i := range infos {
		key := strings.TrimPrefix(infos[i].Key, scope)
		contentType := infos[i].ContentType
		if contentType == "" {
			contentType = contentTypeForKey(key)
		}
		assets = append(assets, &models.Asset{
			Key:          key,
			ContentType:  contentType,
			Size:         infos[i].Size,
			LastModified: infos[i].LastModified,
		})
	}
	return assets, nil
}

// ListFolder returns one level of the app's asset tree: the subfolders directly
// under folderPath (with the count and bytes of everything beneath each) and
// the assets whose key sits exactly there. An empty folderPath is the root;
// otherwise it is the folder with or without a trailing slash.
//
// Object storage has no directories, so the level comes from two sources: a
// delimiter listing of the bucket (one page per request, paged to the end)
// names the direct objects and the prefixes beneath them, and the folder
// bookkeeping rows supply each subfolder's rollup and the explicit folders
// that hold no object yet. The bucket decides what exists; the rows only
// decorate it. A subfolder the bucket knows but the rows do not shows with a
// zero rollup until the next rebuild.
func (s *Assets) ListFolder(ctx context.Context, appID, folderPath string) (*models.AssetLevel, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	folderPath = strings.Trim(folderPath, "/")
	prefix := ""
	if folderPath != "" {
		prefix = folderPath + "/"
	}
	scope := s.objectKey(tenantID, appID, "")
	var objects []grub.ObjectInfo
	var prefixes []string
	cursor := ""
	for {
		page, listErr := s.bucket.ListLevel(ctx, scope+prefix, "/", cursor, 0)
		if listErr != nil {
			return nil, fmt.Errorf("listing asset folder %q: %w", folderPath, listErr)
		}
		objects = append(objects, page.Objects...)
		prefixes = append(prefixes, page.Prefixes...)
		if page.Next == "" {
			break
		}
		cursor = page.Next
	}
	rows, err := s.books.children(ctx, tenantID, appID, folderPath)
	if err != nil {
		return nil, fmt.Errorf("reading asset folder rows %q: %w", folderPath, err)
	}
	return foldLevel(folderPath, scope, objects, prefixes, rows), nil
}

// foldLevel assembles one level from a delimiter listing and the folder rows
// directly beneath it. Every stored key is "<scope><key>"; scope is stripped
// to recover the app-facing key. Subfolders are the union of the bucket's
// common prefixes and the rows (explicit folders may hold no object), each
// carrying its row's rollup when there is one. Folders sort by name, assets by
// key, so the level is stable whatever order the sources returned.
func foldLevel(folderPath, scope string, objects []grub.ObjectInfo, prefixes []string, rows []*models.AssetFolderStat) *models.AssetLevel {
	level := &models.AssetLevel{Path: folderPath, Folders: []models.AssetFolder{}, Assets: []*models.Asset{}}
	for i := range objects {
		key := strings.TrimPrefix(objects[i].Key, scope)
		contentType := objects[i].ContentType
		if contentType == "" {
			contentType = contentTypeForKey(key)
		}
		level.Assets = append(level.Assets, &models.Asset{
			Key:          key,
			ContentType:  contentType,
			Size:         objects[i].Size,
			LastModified: objects[i].LastModified,
		})
	}

	folders := map[string]*models.AssetFolder{}
	for _, p := range prefixes {
		// "<scope><folder>/<name>/" → "<name>".
		name := path.Base(strings.TrimSuffix(strings.TrimPrefix(p, scope), "/"))
		if _, ok := folders[name]; !ok {
			folders[name] = &models.AssetFolder{Name: name}
		}
	}
	for _, row := range rows {
		name := path.Base(row.Path)
		f, ok := folders[name]
		if !ok {
			f = &models.AssetFolder{Name: name}
			folders[name] = f
		}
		f.Count = int(row.Count)
		f.Size = row.Bytes
		f.LastWrittenAt = row.LastWrittenAt
	}
	for _, f := range folders {
		level.Folders = append(level.Folders, *f)
	}
	sort.Slice(level.Folders, func(i, j int) bool { return level.Folders[i].Name < level.Folders[j].Name })
	sort.Slice(level.Assets, func(i, j int) bool { return level.Assets[i].Key < level.Assets[j].Key })
	return level
}

// ErrInvalidAssetPath is returned for a folder path or asset key that is
// empty or has an empty, ".", or ".." segment.
var ErrInvalidAssetPath = errors.New("invalid asset path")

// ErrAssetExists is returned when a move's destination key already holds an
// asset; a move never overwrites.
var ErrAssetExists = errors.New("an asset with that key already exists")

// validFolderPath reports whether p (already trimmed of slashes) names a
// folder: at least one segment, every segment a plain name.
func validFolderPath(p string) bool {
	if p == "" {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

// Move changes an asset's key: into another folder, under another name, or
// both. Folders are key prefixes, so the destination's folder need not
// exist first, and the source folder lingers only if it was explicit or
// still holds something. The destination must be free (ErrAssetExists) and
// a plain path (ErrInvalidAssetPath); a missing source is ErrNotFound.
//
// Object storage has no rename, so the bytes are read and written at the
// new key, then the old object is deleted; the content type carries over.
// The bookkeeping sees one delete and one write in a single transaction.
// It returns the asset at its new key, bytes omitted.
func (s *Assets) Move(ctx context.Context, appID, key, newKey string) (*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	newKey = strings.Trim(newKey, "/")
	if !validFolderPath(newKey) {
		return nil, ErrInvalidAssetPath
	}
	if newKey == key {
		return nil, ErrAssetExists
	}
	src, dst := s.objectKey(tenantID, appID, key), s.objectKey(tenantID, appID, newKey)
	prev, err := s.stat(ctx, src, key)
	if err != nil {
		return nil, err
	}
	if prev == nil {
		return nil, ErrNotFound
	}
	taken, err := s.stat(ctx, dst, newKey)
	if err != nil {
		return nil, err
	}
	if taken != nil {
		return nil, ErrAssetExists
	}
	data, info, err := s.bucket.Get(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("reading asset %q: %w", key, err)
	}
	contentType := prev.ContentType
	if info.ContentType != "" {
		contentType = info.ContentType
	}
	if err := s.bucket.Put(ctx, dst, data, &grub.ObjectInfo{Key: dst, ContentType: contentType, Size: prev.Size}); err != nil {
		return nil, fmt.Errorf("writing asset %q: %w", newKey, err)
	}
	if err := s.bucket.Delete(ctx, src); err != nil && !errors.Is(err, grub.ErrNotFound) {
		return nil, fmt.Errorf("removing moved asset %q: %w", key, err)
	}
	kind := models.KindOf(contentType)
	s.book(ctx, tenantID, appID, newKey,
		deleteChange(key, kind, prev),
		writeChange(newKey, kind, prev.Size, nil),
	)
	events.Asset.Moved.Emit(ctx, events.AssetMovedEvent{
		Key: key, NewKey: newKey, TenantID: tenantID, AppID: appID, ContentType: contentType, Size: prev.Size,
	})
	return &models.Asset{Key: newKey, ContentType: contentType, Size: prev.Size, LastModified: s.books.now()}, nil
}

// CreateFolder makes folderPath an explicit folder of the app: a level that
// exists — and lists — without an object beneath it. Every ancestor becomes
// explicit too, as mkdir -p would, so the new folder is reachable from the
// root. The app must exist for the tenant (ErrNotFound otherwise). Creating a
// folder that already exists, explicitly or by holding objects, is a no-op
// beyond marking it. It returns the folder's level.
func (s *Assets) CreateFolder(ctx context.Context, appID, folderPath string) (*models.AssetLevel, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	folderPath = strings.Trim(folderPath, "/")
	if !validFolderPath(folderPath) {
		return nil, ErrInvalidAssetPath
	}
	if _, err := s.apps.Get(ctx, appID); err != nil {
		return nil, err // ErrNotFound when the app is absent for the tenant
	}
	if err := s.books.markExplicit(ctx, tenantID, appID, folderPath); err != nil {
		return nil, fmt.Errorf("creating asset folder %q: %w", folderPath, err)
	}
	return s.ListFolder(ctx, appID, folderPath)
}

// Delete removes the app's asset at key. ErrNotFound when it does not exist.
func (s *Assets) Delete(ctx context.Context, appID, key string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	obj := s.objectKey(tenantID, appID, key)
	prev, err := s.stat(ctx, obj, key)
	if err != nil {
		return err
	}
	if prev == nil {
		return ErrNotFound
	}
	if err := s.bucket.Delete(ctx, obj); err != nil {
		if errors.Is(err, grub.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("deleting asset %q: %w", key, err)
	}
	s.book(ctx, tenantID, appID, key, deleteChange(key, models.KindOf(prev.ContentType), prev))
	events.Asset.Deleted.Emit(ctx, events.AssetDeletedEvent{
		Key: key, TenantID: tenantID, AppID: appID,
		ContentType: prev.ContentType, Size: prev.Size, LastModified: prev.LastModified,
	})
	return nil
}
