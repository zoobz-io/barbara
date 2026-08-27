//go:build testing

package stores

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/opensearch-project/opensearch-go/v4"
	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/grub"
	grubopensearch "github.com/zoobz-io/grub/opensearch"
	osrenderer "github.com/zoobz-io/lucene/opensearch"
)

// osProvider builds the OpenSearch provider for integration tests, skipping
// when no cluster is reachable — so the suite is a no-op without the dev stack.
func osProvider(t *testing.T) grub.SearchProvider {
	t.Helper()
	addr := env("APP_OPENSEARCH_ADDR", "http://localhost:9200")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+"/_cluster/health", nil)
	if err != nil {
		t.Fatalf("building health request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("OpenSearch not reachable at %s (%v); skipping", addr, err)
	}
	_ = resp.Body.Close()

	client, err := opensearch.NewClient(opensearch.Config{Addresses: []string{addr}})
	if err != nil {
		t.Fatalf("opensearch client: %v", err)
	}
	return grubopensearch.New(client, grubopensearch.Config{Version: osrenderer.V2})
}

// TestSearch_IndexDelete exercises the search store's write side against a real
// cluster: the aggregate is constructed, a projection payload is indexed by
// document ID, then deleted. Covers stores.New and the Index/Delete path the
// jobs pipeline drains through.
func TestSearch_IndexDelete(t *testing.T) {
	provider := osProvider(t)
	db := jobsDB(t) // real Postgres; also does sum.Reset()+New() for the catalog
	st := New(db, astqlpg.New(), provider)
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
