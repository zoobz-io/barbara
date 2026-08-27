package auth

import (
	"context"
	"errors"
	"testing"
)

func TestRequireTenant(t *testing.T) {
	ctx := WithPrincipal(context.Background(), NewPrincipal("u", "tenant-1", "", nil, nil))
	id, err := RequireTenant(ctx)
	if err != nil || id != "tenant-1" {
		t.Fatalf("RequireTenant = (%q, %v), want (tenant-1, nil)", id, err)
	}
	if _, err := RequireTenant(context.Background()); !errors.Is(err, ErrNoTenant) {
		t.Errorf("RequireTenant without principal = %v, want ErrNoTenant", err)
	}
}

func TestRequireUser(t *testing.T) {
	ctx := WithPrincipal(context.Background(), NewPrincipal("user-1", "t", "", nil, nil))
	id, err := RequireUser(ctx)
	if err != nil || id != "user-1" {
		t.Fatalf("RequireUser = (%q, %v), want (user-1, nil)", id, err)
	}
	// A principal with no user id is treated as no user.
	empty := WithPrincipal(context.Background(), NewPrincipal("", "t", "", nil, nil))
	if _, err := RequireUser(empty); !errors.Is(err, ErrNoUser) {
		t.Errorf("RequireUser without user = %v, want ErrNoUser", err)
	}
}
