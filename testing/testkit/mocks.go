//go:build testing

package testkit

import (
	"context"

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
