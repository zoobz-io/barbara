//go:build testing

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// rebuild boots the runtime, rebuilds every app's asset bookkeeping from
// object storage, and tears down. With the dev stack up it returns a
// non-negative count; the test skips when the infra it needs is absent.
func TestRebuild_RebuildsFromStorage(t *testing.T) {
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
