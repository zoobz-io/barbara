//go:build testing

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
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
		addr = "http://localhost:19200"
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
// live OpenSearch and asserts the documents index exists afterward, and that a
// second run is a no-op. The index is left in place: it is the one the dev
// stack's API boots against, and deleting it would open the window a stray
// job-runner write needs to auto-create a dynamically mapped replacement.
func TestEnsureIndices_CreatesDocumentsIndex(t *testing.T) {
	addr := osAddr(t)
	ensureDocumentsIndex(t, addr)
	if !indexExists(t, addr, "documents") {
		t.Fatal("expected documents index to exist after EnsureIndices")
	}

	// Idempotent second run must not error.
	if err := boot.EnsureIndices(context.Background(), addr); err != nil {
		t.Fatalf("EnsureIndices (second run): %v", err)
	}
}

// ensureDocumentsIndex makes the shared documents index exist with the
// explicit mapping. It repairs the index only when something auto-created it
// with a dynamic mapping (ids as text, no analyzer) — which a job runner
// outside the suite does if it writes while no index exists — since the
// reconcile cannot fix that in place. A healthy index is never deleted: the
// cluster is shared with the running dev stack, and tests clean up their own
// documents instead.
func ensureDocumentsIndex(t *testing.T, addr string) {
	t.Helper()
	if indexExists(t, addr, "documents") && !indexHasKeywordKey(t, addr, "documents") {
		t.Logf("documents index has a dynamic mapping (auto-created by a stray write); recreating it")
		deleteIndex(t, addr, "documents")
	}
	if err := boot.EnsureIndices(context.Background(), addr); err != nil {
		t.Fatalf("EnsureIndices: %v", err)
	}
}

// clearDocumentsIndex ensures the shared documents index (repairing a dynamic
// one) and removes every document in it, refreshed, so a fixture starts from
// an empty, explicitly mapped index without ever deleting it. Deleting the
// index is what lets a stray job-runner write auto-create a dynamically
// mapped replacement that the next run cannot reconcile.
func clearDocumentsIndex(t *testing.T, addr string) {
	t.Helper()
	ensureDocumentsIndex(t, addr)
	body := strings.NewReader(`{"query":{"match_all":{}}}`)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		addr+"/documents/_delete_by_query?refresh=true&conflicts=proceed", body)
	if err != nil {
		t.Fatalf("building delete-by-query request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("clearing documents index: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clearing documents index: status %d", resp.StatusCode)
	}
}

// indexHasKeywordKey reports whether the index maps "key" as a keyword — the
// signature of the explicit mapping, which a dynamically created index lacks.
func indexHasKeywordKey(t *testing.T, addr, index string) bool {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, addr+"/"+index+"/_mapping", nil)
	if err != nil {
		t.Fatalf("building mapping request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s mapping: %v", index, err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body map[string]struct {
		Mappings struct {
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding %s mapping: %v", index, err)
	}
	return body[index].Mappings.Properties["key"].Type == "keyword"
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
