//go:build testing

package stores

import (
	"context"
	"testing"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/sum"

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
