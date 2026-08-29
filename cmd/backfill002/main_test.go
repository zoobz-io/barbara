//go:build testing

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// backfill boots the runtime, seeds the 002 tree, and tears down. With the dev
// stack up it returns a result (possibly all zeros — reruns are no-ops); the
// test skips when the infra it needs is absent.
func TestBackfill_BootsAndRuns(t *testing.T) {
	sum.Reset()

	res, err := backfill(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "connecting to database") ||
			strings.Contains(err.Error(), "ensuring indices") ||
			strings.Contains(err.Error(), "connecting to storage") {
			t.Skipf("dev stack not up; skipping: %v", err)
		}
		t.Fatalf("backfill: %v", err)
	}
	if res.Tenants < 0 || res.Documents < 0 {
		t.Errorf("negative counts in result: %+v", res)
	}
}
