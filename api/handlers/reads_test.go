//go:build testing

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockReads is a contracts.Reads whose behavior each test sets.
type mockReads struct {
	doc      *models.DocumentIndex
	list     []models.DocumentIndex
	total    int64
	err      error
	gotApp   string
	gotKey   string
	gotTag   string
	gotPath  string
	gotQuery string
}

func (m *mockReads) GetPublishedByKey(_ context.Context, appID, key string) (*models.DocumentIndex, error) {
	m.gotApp, m.gotKey = appID, key
	return m.doc, m.err
}
func (m *mockReads) Enumerate(_ context.Context, appID, tag string, _, _ int) ([]models.DocumentIndex, int64, error) {
	m.gotApp, m.gotTag = appID, tag
	return m.list, m.total, m.err
}
func (m *mockReads) ListFolder(_ context.Context, appID, parentPath string, _, _ int) ([]models.DocumentIndex, int64, error) {
	m.gotApp, m.gotPath = appID, parentPath
	return m.list, m.total, m.err
}
func (m *mockReads) Search(_ context.Context, appID, query string, _, _ int) ([]models.DocumentIndex, int64, error) {
	m.gotApp, m.gotQuery = appID, query
	return m.list, m.total, m.err
}

// driver wires the site-facing read handlers over the mock contract.
func driver(t *testing.T, mock contracts.Reads) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Reads](k, mock)
	}, All()...)
}

func TestGetPublishedDocument_OK(t *testing.T) {
	mock := &mockReads{doc: &models.DocumentIndex{
		DocumentID: "d1", Key: "guides/install.md", TenantID: "t1", AppID: "app-1", VersionID: "v1",
	}}
	w := driver(t, mock).Request(t, http.MethodGet, "/published/apps/app-1/lookup?key=guides/install.md", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "app-1" || mock.gotKey != "guides/install.md" {
		t.Errorf("store got app=%q key=%q", mock.gotApp, mock.gotKey)
	}
	// Internal fields must not appear in the site-facing response.
	var raw map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if _, ok := raw["tenant_id"]; ok {
		t.Errorf("response leaked tenant_id: %s", w.Body.String())
	}
	if _, ok := raw["version_id"]; ok {
		t.Errorf("response leaked version_id: %s", w.Body.String())
	}
	if raw["document_id"] != "d1" {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

func TestGetPublishedDocument_MissingKey(t *testing.T) {
	w := driver(t, &mockReads{}).Request(t, http.MethodGet, "/published/apps/app-1/lookup", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestGetPublishedDocument_NotFound(t *testing.T) {
	w := driver(t, &mockReads{err: stores.ErrNotFound}).Request(t, http.MethodGet, "/published/apps/app-1/lookup?key=nope.md", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestGetPublishedDocument_NoTenant(t *testing.T) {
	w := driver(t, &mockReads{err: auth.ErrNoTenant}).Request(t, http.MethodGet, "/published/apps/app-1/lookup?key=x.md", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestEnumerateDocuments_OK(t *testing.T) {
	mock := &mockReads{
		list:  []models.DocumentIndex{{DocumentID: "d1"}, {DocumentID: "d2"}},
		total: 2,
	}
	w := driver(t, mock).Request(t, http.MethodGet, "/published/apps/app-1/documents?tag=guide&limit=10", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "app-1" || mock.gotTag != "guide" {
		t.Errorf("store got app=%q tag=%q, want app-1/guide", mock.gotApp, mock.gotTag)
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestListPublishedFolder_OK(t *testing.T) {
	mock := &mockReads{list: []models.DocumentIndex{{DocumentID: "d1"}}, total: 1}
	w := driver(t, mock).Request(t, http.MethodGet, "/published/apps/app-1/folder?path=guides/setup", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "app-1" || mock.gotPath != "guides/setup" {
		t.Errorf("store got app=%q path=%q, want app-1/guides/setup", mock.gotApp, mock.gotPath)
	}
}

func TestListPublishedFolder_RootIsAbsentPath(t *testing.T) {
	mock := &mockReads{list: nil, total: 0}
	w := driver(t, mock).Request(t, http.MethodGet, "/published/apps/app-1/folder", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotPath != "" {
		t.Errorf("absent path resolved to %q, want \"\" (the app root)", mock.gotPath)
	}
}

func TestSearchDocuments_OK(t *testing.T) {
	mock := &mockReads{list: []models.DocumentIndex{{DocumentID: "d1"}}, total: 1}
	w := driver(t, mock).Request(t, http.MethodGet, "/published/apps/app-1/search?q=install", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "app-1" || mock.gotQuery != "install" {
		t.Errorf("store got app=%q query=%q, want app-1/install", mock.gotApp, mock.gotQuery)
	}
}

func TestSearchDocuments_MissingQuery(t *testing.T) {
	w := driver(t, &mockReads{}).Request(t, http.MethodGet, "/published/apps/app-1/search", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}
