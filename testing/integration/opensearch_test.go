//go:build testing

package integration

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/zoobz-io/barbara/internal/boot"
)

// osAddr returns the OpenSearch address for integration tests, and skips the
// test if OpenSearch is not reachable — so the suite is a no-op on machines
// without the dev stack up (e.g. `make test` in CI) and real only when it is.
func osAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("APP_OPENSEARCH_ADDR")
	if addr == "" {
		addr = "http://localhost:9200"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+"/_cluster/health", nil)
	if err != nil {
		t.Fatalf("building health request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		integrationSkip(t, "OpenSearch not reachable at %s (%v)", addr, err)
	}
	_ = resp.Body.Close()
	return addr
}

// TestEnsureIndices_CreatesDocumentsIndex runs the real EnsureIndices against a
// live OpenSearch and asserts the documents index exists afterward. Idempotent:
// safe to re-run, and it cleans up the index it may have created.
func TestEnsureIndices_CreatesDocumentsIndex(t *testing.T) {
	addr := osAddr(t)
	ctx := context.Background()

	preexisting := indexExists(t, addr, "documents")
	if !preexisting {
		t.Cleanup(func() { deleteIndex(t, addr, "documents") })
	}

	if err := boot.EnsureIndices(ctx, addr); err != nil {
		t.Fatalf("EnsureIndices: %v", err)
	}
	if !indexExists(t, addr, "documents") {
		t.Fatal("expected documents index to exist after EnsureIndices")
	}

	// Idempotent second run must not error.
	if err := boot.EnsureIndices(ctx, addr); err != nil {
		t.Fatalf("EnsureIndices (second run): %v", err)
	}
}

func indexExists(t *testing.T, addr, index string) bool {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, addr+"/"+index, nil)
	if err != nil {
		t.Fatalf("building HEAD request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD %s: %v", index, err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func deleteIndex(t *testing.T, addr, index string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, addr+"/"+index, nil)
	if err != nil {
		t.Fatalf("building DELETE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", index, err)
	}
	_ = resp.Body.Close()
}
