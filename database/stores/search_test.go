//go:build testing

package stores

import (
	"context"
	"testing"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/lucene"
	"github.com/zoobz-io/sum"
)

// mockProvider is a grub.SearchProvider that records writes, so search.go's
// logic can be unit-tested without a live cluster.
type mockProvider struct {
	indexed   map[string][]byte
	deleted   []string
	lastIndex string
}

func (m *mockProvider) Index(_ context.Context, index, id string, doc []byte) error {
	if m.indexed == nil {
		m.indexed = map[string][]byte{}
	}
	m.indexed[id] = doc
	m.lastIndex = index
	return nil
}
func (m *mockProvider) Delete(_ context.Context, index, id string) error {
	m.deleted = append(m.deleted, id)
	m.lastIndex = index
	return nil
}
func (*mockProvider) IndexBatch(context.Context, string, map[string][]byte) error { return nil }
func (*mockProvider) Get(context.Context, string, string) ([]byte, error)         { return nil, nil }
func (*mockProvider) DeleteBatch(context.Context, string, []string) error         { return nil }
func (*mockProvider) Exists(context.Context, string, string) (bool, error)        { return false, nil }
func (*mockProvider) Search(context.Context, string, *lucene.Search) (*grub.SearchResponse, error) {
	return nil, nil
}
func (*mockProvider) Count(context.Context, string, lucene.Query) (int64, error) { return 0, nil }
func (*mockProvider) Refresh(context.Context, string) error                      { return nil }

func newTestSearch(p grub.SearchProvider) *Search {
	sum.Reset()
	sum.New()
	return NewSearch(p)
}

// A valid payload is decoded and written to the documents index by document ID.
func TestSearch_Index_DecodesAndWrites(t *testing.T) {
	mp := &mockProvider{}
	s := newTestSearch(mp)

	payload := []byte(`{"document_id":"d1","tenant_id":"t1","key":"a.md","content":"hi","tags":["docs"]}`)
	if err := s.Index(context.Background(), "d1", payload); err != nil {
		t.Fatalf("index: %v", err)
	}
	if _, ok := mp.indexed["d1"]; !ok {
		t.Error("provider.Index was not called for the document")
	}
	if mp.lastIndex != documentIndex {
		t.Errorf("wrote to index %q, want %q", mp.lastIndex, documentIndex)
	}
}

// An invalid payload is rejected before any write reaches the cluster.
func TestSearch_Index_RejectsInvalidPayload(t *testing.T) {
	mp := &mockProvider{}
	s := newTestSearch(mp)

	if err := s.Index(context.Background(), "d1", []byte("not json")); err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
	if len(mp.indexed) != 0 {
		t.Error("provider.Index was called despite a malformed payload")
	}
}

// Delete removes the document by ID from the documents index.
func TestSearch_Delete(t *testing.T) {
	mp := &mockProvider{}
	s := newTestSearch(mp)

	if err := s.Delete(context.Background(), "d1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mp.deleted) != 1 || mp.deleted[0] != "d1" {
		t.Errorf("provider.Delete calls = %v, want [d1]", mp.deleted)
	}
	if mp.lastIndex != documentIndex {
		t.Errorf("deleted from index %q, want %q", mp.lastIndex, documentIndex)
	}
}
