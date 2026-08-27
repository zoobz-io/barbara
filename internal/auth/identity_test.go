package auth

import "testing"

func TestPrincipal_Getters(t *testing.T) {
	p := NewPrincipal("user-1", "tenant-1", "u@example.com", []string{"admin"}, []string{"documents:write"})

	if p.ID() != "user-1" {
		t.Errorf("ID = %q, want user-1", p.ID())
	}
	if p.TenantID() != "tenant-1" {
		t.Errorf("TenantID = %q, want tenant-1", p.TenantID())
	}
	if p.Email() != "u@example.com" {
		t.Errorf("Email = %q, want u@example.com", p.Email())
	}
	if p.Stats() != nil {
		t.Errorf("Stats = %v, want nil", p.Stats())
	}
}

func TestPrincipal_Entitlement(t *testing.T) {
	p := NewPrincipal("u", "t", "", []string{"admin", "editor"}, []string{"documents:write"})

	if !p.HasRole("admin") || !p.HasRole("editor") {
		t.Error("expected admin and editor roles")
	}
	if p.HasRole("owner") {
		t.Error("did not expect owner role")
	}
	if !p.HasScope("documents:write") {
		t.Error("expected documents:write scope")
	}
	if p.HasScope("documents:delete") {
		t.Error("did not expect documents:delete scope")
	}
}

// Construction copies the role/scope slices so later mutation of the caller's
// slice cannot alter the identity.
func TestPrincipal_DefensiveCopy(t *testing.T) {
	roles := []string{"admin"}
	scopes := []string{"documents:write"}
	p := NewPrincipal("u", "t", "", roles, scopes)

	roles[0] = "tampered"
	scopes[0] = "tampered"

	if !p.HasRole("admin") {
		t.Error("role slice was not defensively copied")
	}
	if !p.HasScope("documents:write") {
		t.Error("scope slice was not defensively copied")
	}
}
