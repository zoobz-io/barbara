//go:build testing

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/zoobz-io/sum"
)

// setup wires the whole service short of serving. With the dev stack up it
// boots the runtime, loads config, registers auth, freezes, and builds
// observability; the test asserts a serviceable result and tears it down.
// Skips when the infra it needs is absent.
func TestSetup_WiresService(t *testing.T) {
	sum.Reset()

	svc, port, cleanup, err := setup(context.Background())
	if err != nil {
		if skippable(err) {
			t.Skipf("dev stack not up; skipping: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	if svc == nil {
		t.Error("setup returned a nil service")
	}
	if port <= 0 {
		t.Errorf("serve port = %d, want a positive port", port)
	}
}

// TestDocuments_EndToEnd drives a real HTTP request through the admin router
// into the real store and Postgres: create a document, then list it back. Skips
// without the dev stack.
func TestDocuments_EndToEnd(t *testing.T) {
	sum.Reset()

	svc, _, cleanup, err := setup(context.Background())
	if err != nil {
		if skippable(err) {
			t.Skipf("dev stack not up; skipping: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	router := svc.Engine().Router()
	const tenant = "e2e11111-0000-0000-0000-000000000001"
	t.Cleanup(func() { _, _ = cleanupDB(t).Exec("DELETE FROM documents WHERE tenant_id = $1", tenant) })

	// Create.
	body, _ := json.Marshal(map[string]string{"key": "e2e/doc.md"})
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenant)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var created struct{ ID, Key string }
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == "" || created.Key != "e2e/doc.md" {
		t.Fatalf("unexpected create response: %s", w.Body.String())
	}

	// List returns it.
	lreq := httptest.NewRequest(http.MethodGet, "/documents", nil)
	lreq.Header.Set("X-Tenant-ID", tenant)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, lreq)
	if lw.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", lw.Code, lw.Body.String())
	}
	if !strings.Contains(lw.Body.String(), created.ID) {
		t.Errorf("list did not include the created document: %s", lw.Body.String())
	}
}

// skippable reports whether an Init error means the dev stack is simply absent.
func skippable(err error) bool {
	return strings.Contains(err.Error(), "connecting to database") ||
		strings.Contains(err.Error(), "ensuring indices")
}

// cleanupDB opens a fresh connection for test cleanup, since the runtime's own
// connection is closed by the deferred Shutdown before t.Cleanup runs.
func cleanupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	env := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("APP_DB_HOST", "127.0.0.1"), env("APP_DB_PORT", "5432"),
		env("APP_DB_USER", "barbara"), env("APP_DB_PASSWORD", "barbara"),
		env("APP_DB_NAME", "barbara"))
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("cleanup db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
