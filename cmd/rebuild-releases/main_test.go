//go:build testing

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// rebuild boots the runtime, rebuilds every release's change rows and counts
// from its entries, and tears down. With the dev stack up it returns a
// non-negative count; the test skips when the infra it needs is absent.
func TestRebuild_RebuildsFromEntries(t *testing.T) {
	sum.Reset()

	n, err := rebuild(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "connecting to database") ||
			strings.Contains(err.Error(), "ensuring indices") ||
			strings.Contains(err.Error(), "connecting to storage") {
			t.Skipf("dev stack not up; skipping: %v", err)
		}
		t.Fatalf("rebuild: %v", err)
	}
	if n < 0 {
		t.Errorf("rebuilt count = %d, want >= 0", n)
	}
}
