//go:build testing

// Package testkit is shared unit-test infrastructure: reusable house-interface
// mocks and an HTTP handler harness, so a feature's tests describe behavior
// instead of re-rolling the same setup. Compiled only under the `testing` build
// tag, so it never ships in a binary.
package testkit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/internal/auth"
)

// DefaultTenant is the tenant the harness scopes requests to unless overridden.
const DefaultTenant = "11111111-0000-0000-0000-000000000001"

// Driver fires HTTP requests at a wired surface and returns the recorded
// response.
type Driver struct {
	handler http.Handler
}

// DriverFor wraps an existing handler (e.g. a real binary's router in an
// end-to-end test) so it can be driven with the same request helpers as the
// mock harness.
func DriverFor(handler http.Handler) *Driver {
	return &Driver{handler: handler}
}

// Handlers builds a surface router ready to drive with httptest: a fresh
// registry, the caller's contracts registered via register, the auth stub
// wired, and the given endpoints mounted, then frozen. The same setup serves
// every surface's handler tests.
//
//	d := testkit.Handlers(t, func(k sum.Key) {
//	    sum.Register[contracts.Documents](k, mock)
//	}, handlers.All()...)
//	w := d.Request(t, http.MethodPost, "/documents", body)
func Handlers(t *testing.T, register func(k sum.Key), endpoints ...rocco.Endpoint) *Driver {
	return HandlersAs(t, auth.DefaultStub(), register, endpoints...)
}

// HandlersAs is Handlers with a caller-supplied authenticator, so tests can
// drive endpoints as an identity that lacks a required scope or role and assert
// the 403 gate — the default stub grants everything.
func HandlersAs(t *testing.T, authenticator auth.Authenticator, register func(k sum.Key), endpoints ...rocco.Endpoint) *Driver {
	t.Helper()
	sum.Reset()
	sum.New()
	k := sum.Start()
	register(k)
	engine := rocco.NewEngine()
	auth.Wire(k, engine, authenticator)
	engine.WithHandlers(endpoints...)
	sum.Freeze(k)
	return &Driver{handler: engine.Router()}
}

// Request fires a request scoped to DefaultTenant, JSON-encoding a non-nil body.
func (d *Driver) Request(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	return d.RequestAs(t, DefaultTenant, method, path, body)
}

// RequestRaw fires a request as the given tenant (empty for none) with a raw
// body and explicit content type — for binary endpoints (rocco.RawBody input,
// rocco.Blob output) that the JSON-encoding Request helpers can't drive. On a
// download the raw bytes are in the returned recorder's Body.
func (d *Driver) RequestRaw(t *testing.T, tenant, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, req)
	return w
}

// RequestAs fires a request as the given tenant (empty for none), JSON-encoding
// a non-nil body.
func (d *Driver) RequestAs(t *testing.T, tenant, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, req)
	return w
}
