package stores

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// defaultAssetContentType is used when an upload declares no content type.
const defaultAssetContentType = "application/octet-stream"

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
type Assets struct {
	bucket grub.BucketProvider
	apps   *Apps
}

// NewAssets creates an assets store over the shared object-storage bucket.
func NewAssets(bucket grub.BucketProvider, apps *Apps) *Assets {
	return &Assets{bucket: bucket, apps: apps}
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
	info := &grub.ObjectInfo{Key: obj, ContentType: contentType, Size: int64(len(data))}
	if err := s.bucket.Put(ctx, obj, data, info); err != nil {
		return nil, fmt.Errorf("putting asset %q: %w", key, err)
	}
	events.Asset.Written.Emit(ctx, events.AssetWrittenEvent{
		Key: key, TenantID: tenantID, AppID: appID, ContentType: contentType, Size: int64(len(data)),
	})
	return &models.Asset{Key: key, ContentType: contentType, Size: int64(len(data))}, nil
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
	return &models.Asset{Key: key, ContentType: info.ContentType, Size: info.Size, Data: data}, nil
}

// List returns metadata for the app's assets, without the bytes, scoped by
// object-key prefix. A non-empty keyPrefix narrows the listing to keys under
// it — the folder view, since an asset folder is a key prefix by convention.
func (s *Assets) List(ctx context.Context, appID, keyPrefix string) ([]*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	scope := tenantID + "/" + appID + "/"
	infos, err := s.bucket.List(ctx, scope+keyPrefix, 0)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}
	assets := make([]*models.Asset, 0, len(infos))
	for i := range infos {
		assets = append(assets, &models.Asset{
			Key:         strings.TrimPrefix(infos[i].Key, scope),
			ContentType: infos[i].ContentType,
			Size:        infos[i].Size,
		})
	}
	return assets, nil
}

// Delete removes the app's asset at key. ErrNotFound when it does not exist.
func (s *Assets) Delete(ctx context.Context, appID, key string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if err := s.bucket.Delete(ctx, s.objectKey(tenantID, appID, key)); err != nil {
		if errors.Is(err, grub.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("deleting asset %q: %w", key, err)
	}
	events.Asset.Deleted.Emit(ctx, events.AssetDeletedEvent{Key: key, TenantID: tenantID, AppID: appID})
	return nil
}
