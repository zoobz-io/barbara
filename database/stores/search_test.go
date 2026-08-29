//go:build testing

package stores

import (
	"context"
	"reflect"
	"testing"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/lucene"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

func newTestSearch(p grub.SearchProvider) *Search {
	sum.Reset()
	sum.New()
	return NewSearch(p)
}

// A valid payload is decoded and written to the documents index by document ID.
func TestSearch_Index_DecodesAndWrites(t *testing.T) {
	mp := testkit.NewSearchProvider()
	s := newTestSearch(mp)

	payload := []byte(`{"document_id":"d1","tenant_id":"t1","key":"a.md","content":"hi","tags":["docs"]}`)
	if err := s.Index(context.Background(), "d1", payload); err != nil {
		t.Fatalf("index: %v", err)
	}
	if _, ok := mp.Indexed["d1"]; !ok {
		t.Error("provider.Index was not called for the document")
	}
	if mp.LastIndex != documentIndex {
		t.Errorf("wrote to index %q, want %q", mp.LastIndex, documentIndex)
	}
}

// An invalid payload is rejected before any write reaches the cluster.
func TestSearch_Index_RejectsInvalidPayload(t *testing.T) {
	mp := testkit.NewSearchProvider()
	s := newTestSearch(mp)

	if err := s.Index(context.Background(), "d1", []byte("not json")); err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
	if len(mp.Indexed) != 0 {
		t.Error("provider.Index was called despite a malformed payload")
	}
}

// Every paginated read must carry a deterministic sort ending in the unique
// document_id tiebreaker, so offset paging is stable across requests. A listing
// (filter context) sorts oldest-first; a full-text search sorts by relevance
// first. Without the sort these reads fall back to unstable Lucene doc order.

func searchCtx() context.Context {
	return auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("u-1", "t-1", "", nil, nil))
}

func TestSearch_Enumerate_SortsByCreatedAtThenDocID(t *testing.T) {
	mp := testkit.NewSearchProvider()
	s := newTestSearch(mp)

	if _, _, err := s.Enumerate(searchCtx(), "", 10, 0); err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	got := mp.LastSearch.SortValue()
	want := []lucene.SortField{
		{Field: "created_at", Order: "asc"},
		{Field: "document_id", Order: "asc"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("enumerate sort = %+v, want %+v", got, want)
	}
}

func TestSearch_FullText_SortsByScoreThenDocID(t *testing.T) {
	mp := testkit.NewSearchProvider()
	s := newTestSearch(mp)

	if _, _, err := s.Search(searchCtx(), "hello", 10, 0); err != nil {
		t.Fatalf("search: %v", err)
	}
	want := []lucene.SortField{
		{Field: "_score", Order: "desc"},
		{Field: "document_id", Order: "asc"},
	}
	if got := mp.LastSearch.SortValue(); !reflect.DeepEqual(got, want) {
		t.Errorf("full-text sort = %+v, want %+v", got, want)
	}

	// The cross-tenant admin search paginates too — same stable ordering.
	if _, _, err := s.SearchAll(context.Background(), "hello", 10, 0); err != nil {
		t.Fatalf("search all: %v", err)
	}
	if got := mp.LastSearch.SortValue(); !reflect.DeepEqual(got, want) {
		t.Errorf("SearchAll sort = %+v, want %+v", got, want)
	}
}

// Delete removes the document by ID from the documents index.
func TestSearch_Delete(t *testing.T) {
	mp := testkit.NewSearchProvider()
	s := newTestSearch(mp)

	if err := s.Delete(context.Background(), "d1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(mp.Deleted) != 1 || mp.Deleted[0] != "d1" {
		t.Errorf("provider.Delete calls = %v, want [d1]", mp.Deleted)
	}
	if mp.LastIndex != documentIndex {
		t.Errorf("deleted from index %q, want %q", mp.LastIndex, documentIndex)
	}
}
