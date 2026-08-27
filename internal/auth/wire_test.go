package auth

import (
	"context"
	"testing"

	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"
)

// Wire registers the resolver under the Authenticator contract so it resolves
// out of the sum registry — the property that makes the janus resolver a
// drop-in. sum.Start is a process-global singleton; this is the only test in
// the package that starts it.
func TestWire_RegistersResolverInRegistry(t *testing.T) {
	k := sum.Start()
	Wire(k, rocco.NewEngine(), DefaultStub())
	sum.Freeze(k)

	a, err := sum.Use[Authenticator](context.Background())
	if err != nil {
		t.Fatalf("Authenticator not resolvable from registry after Wire: %v", err)
	}

	id, err := a.Authenticate(context.Background(), httpReq(t, ""))
	if err != nil {
		t.Fatalf("resolved authenticator failed: %v", err)
	}
	if id.TenantID() != DefaultTenantID {
		t.Errorf("TenantID = %q, want %q", id.TenantID(), DefaultTenantID)
	}
}
