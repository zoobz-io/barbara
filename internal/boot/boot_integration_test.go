//go:build testing

package boot

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"
)

// TestInit_WiresRuntime drives the real Init. With Postgres and OpenSearch both
// reachable it boots the whole spine — connections, index bootstrap, stores,
// boundaries, and the jobs runner — and Shutdown tears it back down. Without a
// reachable cluster Init fails at EnsureIndices, which still proves the DB and
// client wiring; without Postgres it skips entirely. So it's a no-op on a bare
// machine and a full end-to-end boot when the dev stack is up.
func TestInit_WiresRuntime(t *testing.T) {
	sum.Reset()

	rt, err := Init(context.Background())
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "connecting to database"):
			t.Skipf("Postgres not reachable; skipping: %v", err)
		case strings.Contains(err.Error(), "ensuring indices"):
			t.Skipf("OpenSearch not reachable; DB/clients wired, skipping full boot: %v", err)
		default:
			t.Fatalf("Init failed with an unexpected wiring error: %v", err)
		}
	}

	// Full boot succeeded — the spine is wired.
	if rt.DB == nil || rt.Svc == nil || rt.Bucket == nil {
		t.Error("Runtime is missing a core connection")
	}
	if rt.Stores == nil || rt.Stores.Jobs == nil || rt.Stores.Search == nil {
		t.Error("stores aggregate is not fully constructed")
	}
	if err := rt.Shutdown(); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}
