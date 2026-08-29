package stores

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zoobz-io/grub"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/internal/auth"
)

// defaultAssetContentType is used when an upload declares no content type.
const defaultAssetContentType = "application/octet-stream"

// Assets is the data-access layer for binary assets in object storage. An asset
// is an opaque blob addressed by a tenant-unique key; there is no Postgres row
// and no versioning — putting the same key overwrites. The backing bucket is
// shared across tenants, so every operation namespaces the stored object key
// with the tenant id: one tenant's keys are invisible to another, and a listing
// only ever sees its own tenant's prefix.
type Assets struct {
	bucket grub.BucketProvider
}

// NewAssets creates an assets store over the shared object-storage bucket.
func NewAssets(bucket grub.BucketProvider) *Assets {
	return &Assets{bucket: bucket}
}

// objectKey namespaces a user-supplied key under the request's tenant. The
// stored object name is "<tenant>/<key>"; List filters by the tenant prefix, so
// a key can never address another tenant's object.
func (s *Assets) objectKey(tenantID, key string) string {
	return tenantID + "/" + key
}

// Put stores data at key for the request's tenant, overwriting any existing
// asset with that key. An empty contentType defaults to octet-stream. It returns
// the stored asset's metadata (bytes omitted) — the normalized content type and
// size the reader will see.
func (s *Assets) Put(ctx context.Context, key, contentType string, data []byte) (*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = defaultAssetContentType
	}
	obj := s.objectKey(tenantID, key)
	info := &grub.ObjectInfo{Key: obj, ContentType: contentType, Size: int64(len(data))}
	if err := s.bucket.Put(ctx, obj, data, info); err != nil {
		return nil, fmt.Errorf("putting asset %q: %w", key, err)
	}
	return &models.Asset{Key: key, ContentType: contentType, Size: int64(len(data))}, nil
}

// Get returns the asset stored at key for the request's tenant, bytes included.
// ErrNotFound when the tenant has no asset with that key.
func (s *Assets) Get(ctx context.Context, key string) (*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	data, info, err := s.bucket.Get(ctx, s.objectKey(tenantID, key))
	if err != nil {
		if errors.Is(err, grub.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting asset %q: %w", key, err)
	}
	return &models.Asset{Key: key, ContentType: info.ContentType, Size: info.Size, Data: data}, nil
}

// List returns metadata for every asset the request's tenant has stored, without
// the bytes, tenant-scoped by object-key prefix.
func (s *Assets) List(ctx context.Context) ([]*models.Asset, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	prefix := tenantID + "/"
	infos, err := s.bucket.List(ctx, prefix, 0)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}
	assets := make([]*models.Asset, 0, len(infos))
	for i := range infos {
		assets = append(assets, &models.Asset{
			Key:         strings.TrimPrefix(infos[i].Key, prefix),
			ContentType: infos[i].ContentType,
			Size:        infos[i].Size,
		})
	}
	return assets, nil
}

// Delete removes the request tenant's asset at key. ErrNotFound when it does not
// exist.
func (s *Assets) Delete(ctx context.Context, key string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if err := s.bucket.Delete(ctx, s.objectKey(tenantID, key)); err != nil {
		if errors.Is(err, grub.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("deleting asset %q: %w", key, err)
	}
	return nil
}
