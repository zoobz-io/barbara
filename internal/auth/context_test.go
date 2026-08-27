package auth

import (
	"context"
	"testing"
)

func TestContext_RoundTrip(t *testing.T) {
	p := NewPrincipal("user-1", "tenant-1", "", []string{"admin"}, nil)
	ctx := WithPrincipal(context.Background(), p)

	got, ok := PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("PrincipalFromContext returned ok=false after WithPrincipal")
	}
	if got.ID() != "user-1" || got.TenantID() != "tenant-1" {
		t.Errorf("round-tripped identity mismatch: id=%q tenant=%q", got.ID(), got.TenantID())
	}
	if TenantFromContext(ctx) != "tenant-1" {
		t.Errorf("TenantFromContext = %q, want tenant-1", TenantFromContext(ctx))
	}
	if UserFromContext(ctx) != "user-1" {
		t.Errorf("UserFromContext = %q, want user-1", UserFromContext(ctx))
	}
}

func TestContext_EmptyContext(t *testing.T) {
	ctx := context.Background()

	if _, ok := PrincipalFromContext(ctx); ok {
		t.Error("PrincipalFromContext should return ok=false on a bare context")
	}
	if TenantFromContext(ctx) != "" {
		t.Errorf("TenantFromContext on bare context = %q, want empty", TenantFromContext(ctx))
	}
	if UserFromContext(ctx) != "" {
		t.Errorf("UserFromContext on bare context = %q, want empty", UserFromContext(ctx))
	}
}
