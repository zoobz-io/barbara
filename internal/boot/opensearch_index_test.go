package boot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEnsureIndices_CreatesWhenMissing(t *testing.T) {
	var mu sync.Mutex
	created := map[string]bool{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := r.URL.Path[1:] // strip leading /
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			mu.Lock()
			created[index] = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	if err := EnsureIndices(context.Background(), srv.URL); err != nil {
		t.Fatalf("EnsureIndices returned error: %v", err)
	}

	if !created["documents"] {
		t.Error("expected index \"documents\" to be created")
	}
}

func TestEnsureIndices_ReconcilesExisting(t *testing.T) {
	var mu sync.Mutex
	var putPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusOK) // everything already exists
		case http.MethodPut:
			mu.Lock()
			putPaths = append(putPaths, r.URL.Path)
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	if err := EnsureIndices(context.Background(), srv.URL); err != nil {
		t.Fatalf("EnsureIndices returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// An existing index is reconciled via PUT /{index}/_mapping (additive), never
	// recreated via a bare PUT /{index}.
	for _, p := range putPaths {
		if p == "/documents" {
			t.Errorf("existing index was recreated via %q, want a _mapping reconcile", p)
		}
	}
	want := "/documents/_mapping"
	found := false
	for _, p := range putPaths {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a PUT to %q, got paths %v", want, putPaths)
	}
}

func TestEnsureIndices_PropagatesReconcileFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusOK) // exists → reconcile path
		case http.MethodPut:
			w.WriteHeader(http.StatusBadRequest) // e.g. a conflicting field type
			_, _ = w.Write([]byte(`{"error":"mapper_parsing_exception"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	if err := EnsureIndices(context.Background(), srv.URL); err == nil {
		t.Error("expected EnsureIndices to return an error on reconcile failure")
	}
}

func TestEnsureIndices_PropagatesCreateFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad mapping"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	if err := EnsureIndices(context.Background(), srv.URL); err == nil {
		t.Error("expected EnsureIndices to return an error on create failure")
	}
}
