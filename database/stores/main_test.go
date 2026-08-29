//go:build testing

package stores

import (
	"os"
	"testing"

	"github.com/zoobz-io/capitan"
)

// TestMain puts capitan in synchronous dispatch mode so domain-event emission
// assertions are deterministic — an event fired by a store method is delivered
// to its listener before the method returns. Configure runs before the default
// capitan instance is created (first Emit/Hook), so the mode takes effect.
func TestMain(m *testing.M) {
	capitan.Configure(capitan.WithSyncMode())
	os.Exit(m.Run())
}
