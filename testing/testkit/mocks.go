//go:build testing

package testkit

import (
	"context"
	"strings"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/lucene"
)

// SearchProvider is a recording grub.SearchProvider for unit-testing search
// stores without a live cluster. Indexed and Deleted capture the writes; the
// Err fields inject failures.
type SearchProvider struct {
	Indexed    map[string][]byte // id -> document bytes
	LastSearch *lucene.Search    // the last search executed (query, paging, sort)
	IndexErr   error
	DeleteErr  error
	LastIndex  string   // the index name last written to
	Deleted    []string // ids deleted
}

// NewSearchProvider returns a ready-to-use recording provider.
func NewSearchProvider() *SearchProvider {
	return &SearchProvider{Indexed: map[string][]byte{}}
}

func (m *SearchProvider) Index(_ context.Context, index, id string, doc []byte) error {
	if m.IndexErr != nil {
		return m.IndexErr
	}
	m.Indexed[id] = doc
	m.LastIndex = index
	return nil
}

func (m *SearchProvider) Delete(_ context.Context, index, id string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.Deleted = append(m.Deleted, id)
	m.LastIndex = index
	return nil
}

func (*SearchProvider) IndexBatch(context.Context, string, map[string][]byte) error { return nil }
func (*SearchProvider) Get(context.Context, string, string) ([]byte, error)         { return nil, nil }
func (*SearchProvider) DeleteBatch(context.Context, string, []string) error         { return nil }
func (*SearchProvider) Exists(context.Context, string, string) (bool, error)        { return false, nil }
func (m *SearchProvider) Search(_ context.Context, _ string, search *lucene.Search) (*grub.SearchResponse, error) {
	m.LastSearch = search
	return &grub.SearchResponse{}, nil
}
func (*SearchProvider) Count(context.Context, string, lucene.Query) (int64, error) { return 0, nil }
func (*SearchProvider) Refresh(context.Context, string) error                      { return nil }

// BucketProvider is an in-memory grub.BucketProvider for unit-testing
// bucket-backed stores without object storage. Objects are keyed by their full
// stored name (the store prepends the tenant), so overwrite-on-same-key and
// tenant isolation by prefix are directly observable.
type BucketProvider struct {
	Objects map[string]BucketObject
}

// BucketObject is a stored blob and its metadata.
type BucketObject struct {
	Data []byte
	Info grub.ObjectInfo
}

// NewBucketProvider returns a ready-to-use in-memory bucket.
func NewBucketProvider() *BucketProvider {
	return &BucketProvider{Objects: map[string]BucketObject{}}
}

// Get returns the blob at key, or grub.ErrNotFound.
func (m *BucketProvider) Get(_ context.Context, key string) ([]byte, *grub.ObjectInfo, error) {
	obj, ok := m.Objects[key]
	if !ok {
		return nil, nil, grub.ErrNotFound
	}
	info := obj.Info
	return obj.Data, &info, nil
}

// Put stores data at key, overwriting any existing object.
func (m *BucketProvider) Put(_ context.Context, key string, data []byte, info *grub.ObjectInfo) error {
	stored := grub.ObjectInfo{}
	if info != nil {
		stored = *info
	}
	stored.Key = key
	stored.Size = int64(len(data))
	m.Objects[key] = BucketObject{Data: append([]byte(nil), data...), Info: stored}
	return nil
}

// Delete removes the blob at key, or returns grub.ErrNotFound.
func (m *BucketProvider) Delete(_ context.Context, key string) error {
	if _, ok := m.Objects[key]; !ok {
		return grub.ErrNotFound
	}
	delete(m.Objects, key)
	return nil
}

// Exists reports whether key is present.
func (m *BucketProvider) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.Objects[key]
	return ok, nil
}

// List returns object info for every key under prefix (limit 0 = no limit).
func (m *BucketProvider) List(_ context.Context, prefix string, limit int) ([]grub.ObjectInfo, error) {
	var out []grub.ObjectInfo
	for key, obj := range m.Objects {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		out = append(out, obj.Info)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
