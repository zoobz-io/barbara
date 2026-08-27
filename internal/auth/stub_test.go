package auth

import (
	"context"
	"net/http"
	"testing"
)

func TestStub_ResolvesDefaultIdentity(t *testing.T) {
	id, err := DefaultStub().Authenticate(context.Background(), httpReq(t, ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.TenantID() != DefaultTenantID {
		t.Errorf("TenantID = %q, want %q", id.TenantID(), DefaultTenantID)
	}
	if id.ID() != DefaultUserID {
		t.Errorf("ID = %q, want %q", id.ID(), DefaultUserID)
	}
	if !id.HasRole(RoleAdmin) {
		t.Errorf("default stub identity should hold the %q role", RoleAdmin)
	}
}

func TestStub_TenantHeaderOverride(t *testing.T) {
	id, err := DefaultStub().Authenticate(context.Background(), httpReq(t, "acme"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.TenantID() != "acme" {
		t.Errorf("TenantID = %q, want acme (from X-Tenant-ID)", id.TenantID())
	}
	// The acting user is unaffected by the tenant override.
	if id.ID() != DefaultUserID {
		t.Errorf("ID = %q, want %q", id.ID(), DefaultUserID)
	}
}

func TestStub_InjectedIdentity(t *testing.T) {
	stub := NewStub("svc-1", "tenant-9", "svc@x.io", []string{"editor"}, []string{"documents:read"})
	id, err := stub.Authenticate(context.Background(), httpReq(t, ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.ID() != "svc-1" || id.TenantID() != "tenant-9" || id.Email() != "svc@x.io" {
		t.Errorf("unexpected identity: id=%q tenant=%q email=%q", id.ID(), id.TenantID(), id.Email())
	}
	if !id.HasScope("documents:read") || id.HasRole(RoleAdmin) {
		t.Error("injected entitlement not honored")
	}
}

func TestStub_EmptyIdentityFallsBackToDefaults(t *testing.T) {
	stub := NewStub("", "", "", nil, nil)
	id, err := stub.Authenticate(context.Background(), httpReq(t, ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.ID() != DefaultUserID || id.TenantID() != DefaultTenantID {
		t.Errorf("empty ids should fall back to defaults, got id=%q tenant=%q", id.ID(), id.TenantID())
	}
}

// httpReq builds a request, optionally carrying an X-Tenant-ID header.
func httpReq(t *testing.T, tenant string) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if tenant != "" {
		r.Header.Set("X-Tenant-ID", tenant)
	}
	return r
}
