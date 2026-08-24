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

func TestEnsureIndices_SkipsExisting(t *testing.T) {
	var mu sync.Mutex
	puts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusOK) // everything already exists
		case http.MethodPut:
			mu.Lock()
			puts++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
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
	if puts != 0 {
		t.Errorf("expected no index creation when all exist, got %d PUTs", puts)
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
