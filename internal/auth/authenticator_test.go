package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/zoobz-io/rocco"
)

// fakeAuthenticator lets the extractor tests drive both outcomes.
type fakeAuthenticator struct {
	id  rocco.Identity
	err error
}

func (f fakeAuthenticator) Authenticate(context.Context, *http.Request) (rocco.Identity, error) {
	return f.id, f.err
}

func TestNewExtractor_ReturnsIdentity(t *testing.T) {
	want := NewPrincipal("u", "t", "", nil, nil)
	extract := NewExtractor(fakeAuthenticator{id: want})

	got, err := extract(context.Background(), httpReq(t, ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("extractor returned a different identity than the authenticator produced")
	}
}

func TestNewExtractor_PropagatesError(t *testing.T) {
	sentinel := errors.New("session invalid")
	extract := NewExtractor(fakeAuthenticator{err: sentinel})

	if _, err := extract(context.Background(), httpReq(t, "")); !errors.Is(err, sentinel) {
		t.Errorf("extractor error = %v, want %v", err, sentinel)
	}
}

// The default stub, adapted through the extractor, resolves a request the way
// rocco's engine will call it.
func TestNewExtractor_WithStub(t *testing.T) {
	extract := NewExtractor(DefaultStub())

	id, err := extract(context.Background(), httpReq(t, "acme"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.TenantID() != "acme" {
		t.Errorf("TenantID = %q, want acme", id.TenantID())
	}
}
