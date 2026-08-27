//go:build testing

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lib/pq"
	"github.com/zoobz-io/rocco"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/admin/contracts"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

// mockDocuments is a contracts.Documents whose behavior each test sets.
type mockDocuments struct {
	doc     *models.Document
	list    []*models.Document
	err     error
	gotKey  string
	deleted bool
}

func (m *mockDocuments) Create(_ context.Context, key string) (*models.Document, error) {
	m.gotKey = key
	return m.doc, m.err
}
func (m *mockDocuments) Get(context.Context, string) (*models.Document, error) {
	return m.doc, m.err
}
func (m *mockDocuments) List(context.Context, int, int) ([]*models.Document, error) {
	return m.list, m.err
}
func (m *mockDocuments) Rename(_ context.Context, _, key string) (*models.Document, error) {
	m.gotKey = key
	return m.doc, m.err
}
func (m *mockDocuments) Delete(context.Context, string) error {
	m.deleted = m.err == nil
	return m.err
}

// router wires the admin handlers over the mock contract with the auth stub,
// returning an http.Handler to drive with httptest.
func router(t *testing.T, mock contracts.Documents) http.Handler {
	t.Helper()
	sum.Reset()
	sum.New()
	k := sum.Start()
	sum.Register[contracts.Documents](k, mock)
	engine := rocco.NewEngine()
	auth.Wire(k, engine, auth.DefaultStub())
	engine.WithHandlers(All()...)
	sum.Freeze(k)
	return engine.Router()
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestCreateDocument_OK(t *testing.T) {
	mock := &mockDocuments{doc: &models.Document{ID: "d1", Key: "guides/a.md", TenantID: "tenant-1"}}
	w := do(t, router(t, mock), http.MethodPost, "/documents", map[string]string{"key": "guides/a.md"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotKey != "guides/a.md" {
		t.Errorf("store got key %q", mock.gotKey)
	}
	var resp struct{ Key, ID string }
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "d1" || resp.Key != "guides/a.md" {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	w := do(t, router(t, &mockDocuments{err: stores.ErrNotFound}), http.MethodGet, "/documents/x", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteDocument_PublishedConflict(t *testing.T) {
	w := do(t, router(t, &mockDocuments{err: stores.ErrDocumentPublished}), http.MethodDelete, "/documents/x", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateDocument_DuplicateConflict(t *testing.T) {
	mock := &mockDocuments{err: &pq.Error{Code: "23505"}}
	w := do(t, router(t, mock), http.MethodPost, "/documents", map[string]string{"key": "dup.md"})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestListDocuments_OK(t *testing.T) {
	mock := &mockDocuments{list: []*models.Document{
		{ID: "d1", Key: "a.md"}, {ID: "d2", Key: "b.md"},
	}}
	w := do(t, router(t, mock), http.MethodGet, "/documents?limit=10&offset=0", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestDeleteDocument_OK(t *testing.T) {
	mock := &mockDocuments{}
	w := do(t, router(t, mock), http.MethodDelete, "/documents/d1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if !mock.deleted {
		t.Error("delete was not called")
	}
}
