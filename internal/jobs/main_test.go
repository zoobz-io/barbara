package jobs

import (
	"os"
	"testing"

	"github.com/zoobz-io/capitan"
)

// TestMain puts capitan in synchronous dispatch so the pipeline's index-write
// events are delivered before Process returns, making the emission assertions
// deterministic.
func TestMain(m *testing.M) {
	capitan.Configure(capitan.WithSyncMode())
	os.Exit(m.Run())
}
