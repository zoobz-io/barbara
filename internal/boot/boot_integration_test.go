//go:build testing

package boot

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// TestInit_ConnectsThenEnsuresIndices drives the real Init far enough to prove
// the wiring: shared config loads, Postgres connects, and the object-storage
// and OpenSearch clients construct. With no OpenSearch reachable, Init fails at
// EnsureIndices — which is exactly the boundary this asserts. Skips when
// Postgres itself is absent, so it's a no-op without the dev stack.
func TestInit_ConnectsThenEnsuresIndices(t *testing.T) {
	sum.Reset()

	rt, err := Init(context.Background())
	if err == nil {
		// OpenSearch was reachable too — full boot succeeded.
		_ = rt.Shutdown()
		return
	}

	switch {
	case strings.Contains(err.Error(), "connecting to database"):
		t.Skipf("Postgres not reachable; skipping: %v", err)
	case strings.Contains(err.Error(), "ensuring indices"):
		// Reached EnsureIndices: config loaded, DB connected, clients built.
	default:
		t.Fatalf("Init failed before reaching indices — wiring bug: %v", err)
	}
}
