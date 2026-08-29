//go:build testing

package integration

import (
	"context"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// TestSearch_IndexDelete exercises the search store's write side against a real
// cluster: the aggregate is constructed, a projection payload is indexed by
// document ID, then deleted. Covers stores.New and the Index/Delete path the
// jobs pipeline drains through.
func TestSearch_IndexDelete(t *testing.T) {
	provider := osProvider(t)
	db := pgDB(t) // real Postgres; pgDB also does sum.Reset()+New() for the catalog
	t.Cleanup(func() { _ = db.Close() })
	st := stores.New(db, astqlpg.New(), provider, testkit.NewBucketProvider())
	ctx := context.Background()

	const id = "doc-int-0000-0000-0000-000000000001"
	payload := []byte(`{"document_id":"` + id + `","tenant_id":"t1","key":"guides/a.md",` +
		`"version_id":"v1","version_number":1,"content":"hello","tags":["docs"]}`)

	if err := st.Search.Index(ctx, id, payload); err != nil {
		t.Fatalf("index: %v", err)
	}
	t.Cleanup(func() { _ = st.Search.Delete(context.Background(), id) })

	if err := st.Search.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
