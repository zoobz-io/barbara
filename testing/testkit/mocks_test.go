//go:build testing

package testkit

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/lucene"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

// The streams behave like Put and Get: PutStream stores what the reader
// yields, stamped from Now, and GetStream reads it back. A failing reader
// stores nothing; a missing key is ErrNotFound.
func TestBucketProvider_Streams(t *testing.T) {
	ctx := context.Background()
	b := NewBucketProvider()
	at := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	b.Now = func() time.Time { return at }

	if err := b.PutStream(ctx, "k", strings.NewReader("hello"), &grub.ObjectInfo{ContentType: "text/plain"}); err != nil {
		t.Fatalf("PutStream: %v", err)
	}
	rc, info, err := b.GetStream(ctx, "k")
	if err != nil {
		t.Fatalf("GetStream: %v", err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(data) != "hello" || info.ContentType != "text/plain" || info.Size != 5 || !info.LastModified.Equal(at) {
		t.Errorf("stream round trip = %q %+v", data, info)
	}
	if _, _, err := b.GetStream(ctx, "missing"); !errors.Is(err, grub.ErrNotFound) {
		t.Errorf("GetStream of a missing key = %v, want ErrNotFound", err)
	}
	if err := b.PutStream(ctx, "bad", failingReader{}, nil); err == nil {
		t.Error("PutStream from a failing reader succeeded")
	}
	if _, ok := b.Objects["bad"]; ok {
		t.Error("a failed PutStream stored an object")
	}
}

// Stat reports metadata without the bytes, Exists and Delete see the same
// objects, and a missing key is ErrNotFound for both reads and deletes.
func TestBucketProvider_StatExistsDelete(t *testing.T) {
	ctx := context.Background()
	b := NewBucketProvider()
	if err := b.Put(ctx, "a", []byte("xyz"), nil); err != nil {
		t.Fatal(err)
	}
	info, err := b.Stat(ctx, "a")
	if err != nil || info.Key != "a" || info.Size != 3 || info.LastModified.IsZero() {
		t.Errorf("Stat = %+v, %v", info, err)
	}
	if _, err := b.Stat(ctx, "nope"); !errors.Is(err, grub.ErrNotFound) {
		t.Errorf("Stat of a missing key = %v, want ErrNotFound", err)
	}
	if ok, err := b.Exists(ctx, "a"); !ok || err != nil {
		t.Errorf("Exists(a) = %v, %v", ok, err)
	}
	if err := b.Delete(ctx, "nope"); !errors.Is(err, grub.ErrNotFound) {
		t.Errorf("Delete of a missing key = %v, want ErrNotFound", err)
	}
	if err := b.Delete(ctx, "a"); err != nil {
		t.Errorf("Delete: %v", err)
	}
	if ok, _ := b.Exists(ctx, "a"); ok {
		t.Error("a deleted key still exists")
	}
}

// Listings come back in key order: List caps at limit, ListPage pages with a
// start-after cursor and reports the last key of a full page as the next
// cursor, and ListLevel folds one level of prefixes and objects.
func TestBucketProvider_Listing(t *testing.T) {
	ctx := context.Background()
	b := NewBucketProvider()
	for _, k := range []string{"t/b/1", "t/a/2", "t/a/1", "t/a/3", "t/root.txt", "u/x"} {
		if err := b.Put(ctx, k, []byte("x"), nil); err != nil {
			t.Fatal(err)
		}
	}
	keys := func(infos []grub.ObjectInfo) []string {
		out := make([]string, 0, len(infos))
		for _, i := range infos {
			out = append(out, i.Key)
		}
		return out
	}

	page, next, err := b.ListPage(ctx, "t/", "", 2)
	if err != nil || !reflect.DeepEqual(keys(page), []string{"t/a/1", "t/a/2"}) || next != "t/a/2" {
		t.Errorf("page 1 = %v next=%q err=%v", keys(page), next, err)
	}
	page, next, _ = b.ListPage(ctx, "t/", next, 2)
	if !reflect.DeepEqual(keys(page), []string{"t/a/3", "t/b/1"}) || next != "t/b/1" {
		t.Errorf("page 2 = %v next=%q", keys(page), next)
	}
	page, next, _ = b.ListPage(ctx, "t/", next, 2)
	if !reflect.DeepEqual(keys(page), []string{"t/root.txt"}) || next != "" {
		t.Errorf("page 3 = %v next=%q", keys(page), next)
	}
	if all, next, _ := b.ListPage(ctx, "t/", "", 0); len(all) != 5 || next != "" {
		t.Errorf("unlimited page = %v next=%q", keys(all), next)
	}
	if limited, _ := b.List(ctx, "t/", 3); !reflect.DeepEqual(keys(limited), []string{"t/a/1", "t/a/2", "t/a/3"}) {
		t.Errorf("List limit 3 = %v", keys(limited))
	}

	level, err := b.ListLevel(ctx, "t/", "/", "", 0)
	if err != nil {
		t.Fatalf("ListLevel: %v", err)
	}
	if !reflect.DeepEqual(level.Prefixes, []string{"t/a/", "t/b/"}) || !reflect.DeepEqual(keys(level.Objects), []string{"t/root.txt"}) || level.Next != "" {
		t.Errorf("ListLevel = prefixes %v objects %v next %q", level.Prefixes, keys(level.Objects), level.Next)
	}
}

// The search provider records writes and the last search, injects the
// configured failures, and answers the rest as no-ops.
func TestSearchProvider(t *testing.T) {
	ctx := context.Background()
	s := NewSearchProvider()

	if err := s.Index(ctx, "docs", "1", []byte("{}")); err != nil {
		t.Fatalf("Index: %v", err)
	}
	if s.LastIndex != "docs" || string(s.Indexed["1"]) != "{}" {
		t.Errorf("Index recorded index %q docs %v", s.LastIndex, s.Indexed)
	}
	if err := s.Delete(ctx, "docs", "1"); err != nil || !reflect.DeepEqual(s.Deleted, []string{"1"}) {
		t.Errorf("Delete recorded %v, err %v", s.Deleted, err)
	}
	search := &lucene.Search{}
	if resp, err := s.Search(ctx, "docs", search); err != nil || resp == nil || s.LastSearch != search {
		t.Errorf("Search = %v, %v; recorded %v", resp, err, s.LastSearch)
	}

	s.IndexErr, s.DeleteErr = errors.New("index down"), errors.New("delete down")
	if err := s.Index(ctx, "docs", "2", nil); err == nil || len(s.Indexed) != 1 {
		t.Errorf("Index with IndexErr = %v, indexed %v", err, s.Indexed)
	}
	if err := s.Delete(ctx, "docs", "2"); err == nil || len(s.Deleted) != 1 {
		t.Errorf("Delete with DeleteErr = %v, deleted %v", err, s.Deleted)
	}

	if err := s.IndexBatch(ctx, "docs", nil); err != nil {
		t.Errorf("IndexBatch: %v", err)
	}
	if got, err := s.Get(ctx, "docs", "1"); got != nil || err != nil {
		t.Errorf("Get = %v, %v", got, err)
	}
	if err := s.DeleteBatch(ctx, "docs", nil); err != nil {
		t.Errorf("DeleteBatch: %v", err)
	}
	if ok, err := s.Exists(ctx, "docs", "1"); ok || err != nil {
		t.Errorf("Exists = %v, %v", ok, err)
	}
	if n, err := s.Count(ctx, "docs", nil); n != 0 || err != nil {
		t.Errorf("Count = %d, %v", n, err)
	}
	if err := s.Refresh(ctx, "docs"); err != nil {
		t.Errorf("Refresh: %v", err)
	}
}
