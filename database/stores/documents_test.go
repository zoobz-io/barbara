package stores

import (
	"context"
	"errors"
	"testing"

	"github.com/zoobz-io/barbara/internal/auth"
)

// tenant is the scoping guard every documents method runs through: it pulls the
// tenant from the request context and refuses without one.
func TestDocuments_tenant(t *testing.T) {
	ctx := auth.WithPrincipal(context.Background(), auth.NewPrincipal("u", "tenant-1", "", nil, nil))
	id, err := tenant(ctx)
	if err != nil {
		t.Fatalf("tenant with principal: %v", err)
	}
	if id != "tenant-1" {
		t.Errorf("tenant = %q, want tenant-1", id)
	}

	if _, err := tenant(context.Background()); !errors.Is(err, ErrNoTenant) {
		t.Errorf("tenant without principal = %v, want ErrNoTenant", err)
	}
}
