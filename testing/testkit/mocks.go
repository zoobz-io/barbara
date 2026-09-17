//go:build testing

package testkit

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"time"

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
// tenant isolation by prefix are directly observable. Listings are returned
// in key order, and every write stamps LastModified from Now (time.Now by
// default; set Now to make the stamp deterministic).
type BucketProvider struct {
	Objects map[string]BucketObject
	Now     func() time.Time
}

// BucketObject is a stored blob and its metadata.
type BucketObject struct {
	Info grub.ObjectInfo
	Data []byte
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
	stored.LastModified = m.now()
	m.Objects[key] = BucketObject{Data: append([]byte(nil), data...), Info: stored}
	return nil
}

// PutStream reads r to the end and stores it at key, like Put.
func (m *BucketProvider) PutStream(ctx context.Context, key string, r io.Reader, info *grub.ObjectInfo) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return m.Put(ctx, key, data, info)
}

// GetStream returns a reader over the blob at key, or grub.ErrNotFound.
func (m *BucketProvider) GetStream(ctx context.Context, key string) (io.ReadCloser, *grub.ObjectInfo, error) {
	data, info, err := m.Get(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), info, nil
}

// Stat returns the metadata at key without the bytes, or grub.ErrNotFound.
func (m *BucketProvider) Stat(_ context.Context, key string) (*grub.ObjectInfo, error) {
	obj, ok := m.Objects[key]
	if !ok {
		return nil, grub.ErrNotFound
	}
	info := obj.Info
	return &info, nil
}

func (m *BucketProvider) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

// keysUnder returns the stored keys under prefix, sorted, so listings are
// deterministic and key-cursor paging is well defined.
func (m *BucketProvider) keysUnder(prefix string) []string {
	keys := make([]string, 0, len(m.Objects))
	for key := range m.Objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
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

// List returns object info for every key under prefix, in key order (limit
// 0 = no limit).
func (m *BucketProvider) List(_ context.Context, prefix string, limit int) ([]grub.ObjectInfo, error) {
	keys := m.keysUnder(prefix)
	out := make([]grub.ObjectInfo, 0, len(keys))
	for _, key := range keys {
		out = append(out, m.Objects[key].Info)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ListPage returns one page of keys under prefix in key order. The cursor is
// the last key of the previous page (start-after), as the minio provider
// does; limit 0 returns everything in one page.
func (m *BucketProvider) ListPage(_ context.Context, prefix, cursor string, limit int) ([]grub.ObjectInfo, string, error) {
	keys := m.keysUnder(prefix)
	out := make([]grub.ObjectInfo, 0, len(keys))
	for _, key := range keys {
		if cursor != "" && key <= cursor {
			continue
		}
		if limit > 0 && len(out) == limit {
			return out, out[len(out)-1].Key, nil
		}
		out = append(out, m.Objects[key].Info)
	}
	return out, "", nil
}

// ListLevel returns the objects and common prefixes directly under prefix,
// folded from the sorted listing with grub.FoldLevel. The whole level comes
// back in one page: cursor and limit are accepted but never truncate, and
// Next is always empty.
func (m *BucketProvider) ListLevel(ctx context.Context, prefix, delimiter, _ string, _ int) (*grub.Level, error) {
	infos, err := m.List(ctx, prefix, 0)
	if err != nil {
		return nil, err
	}
	return grub.FoldLevel(prefix, delimiter, infos), nil
}
