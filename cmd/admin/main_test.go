//go:build testing

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// setup wires the whole service short of serving. With the dev stack up it
// boots the runtime, loads config, registers auth, freezes, and builds
// observability; the test asserts a serviceable result and tears it down.
// Skips when the infra it needs is absent.
//
// The admin surface no longer serves tenant-scoped authoring — that folded onto
// the public API — so this only asserts the binary boots. The cross-tenant
// SearchAll route is covered by its handler tests.
func TestSetup_WiresService(t *testing.T) {
	sum.Reset()

	svc, port, cleanup, err := setup(context.Background())
	if err != nil {
		if skippable(err) {
			t.Skipf("dev stack not up; skipping: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	if svc == nil {
		t.Error("setup returned a nil service")
	}
	if port <= 0 {
		t.Errorf("serve port = %d, want a positive port", port)
	}
}

// skippable reports whether an Init error means the dev stack is simply absent.
func skippable(err error) bool {
	return strings.Contains(err.Error(), "connecting to database") ||
		strings.Contains(err.Error(), "ensuring indices")
}
