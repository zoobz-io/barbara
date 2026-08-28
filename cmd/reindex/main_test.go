//go:build testing

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// reindex boots the runtime, rebuilds the index from Postgres, and tears down.
// With the dev stack up it returns a non-negative count; the test skips when the
// infra it needs is absent.
func TestReindex_RebuildsFromPostgres(t *testing.T) {
	sum.Reset()

	n, err := reindex(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "connecting to database") ||
			strings.Contains(err.Error(), "ensuring indices") ||
			strings.Contains(err.Error(), "connecting to storage") {
			t.Skipf("dev stack not up; skipping: %v", err)
		}
		t.Fatalf("reindex: %v", err)
	}
	if n < 0 {
		t.Errorf("reindexed count = %d, want >= 0", n)
	}
}
