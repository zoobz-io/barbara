//go:build testing

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lib/pq"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockDocuments is a contracts.Documents whose behavior each test sets.
type mockDocuments struct {
	doc     *models.Document
	head    *models.DocumentHead
	list    []*models.Document
	err     error
	gotKey  string
	gotTag  string
	deleted bool
}

func (m *mockDocuments) Create(_ context.Context, key string) (*models.Document, error) {
	m.gotKey = key
	return m.doc, m.err
}
func (m *mockDocuments) Get(context.Context, string) (*models.Document, error) {
	return m.doc, m.err
}
func (m *mockDocuments) GetWithHead(context.Context, string) (*models.DocumentHead, error) {
	return m.head, m.err
}
func (m *mockDocuments) List(context.Context, int, int) ([]*models.Document, error) {
	return m.list, m.err
}
func (m *mockDocuments) ListByTag(_ context.Context, tag string, _, _ int) ([]*models.Document, error) {
	m.gotTag = tag
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

// docDriver wires the document handlers over the mock contract.
func docDriver(t *testing.T, mock contracts.Documents) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Documents](k, mock)
	}, All()...)
}

func TestCreateDocument_OK(t *testing.T) {
	mock := &mockDocuments{doc: &models.Document{ID: "d1", Key: "guides/a.md", TenantID: "tenant-1"}}
	w := docDriver(t, mock).Request(t, http.MethodPost, "/documents", map[string]string{"key": "guides/a.md"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotKey != "guides/a.md" {
		t.Errorf("store got key %q", mock.gotKey)
	}
	var resp struct {
		Key, ID, Status string
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "d1" || resp.Key != "guides/a.md" || resp.Status != "draft" {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

// GetDocument carries lifecycle status, derived from the published pointer.
func TestGetDocument_OK_WithStatus(t *testing.T) {
	pv := "v2"
	mock := &mockDocuments{doc: &models.Document{ID: "d1", Key: "a.md", PublishedVersionID: &pv}}
	w := docDriver(t, mock).Request(t, http.MethodGet, "/documents/d1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "d1" || resp.Status != "published" {
		t.Errorf("get response = %s, want d1/published", w.Body.String())
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	w := docDriver(t, &mockDocuments{err: stores.ErrNotFound}).Request(t, http.MethodGet, "/documents/x", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// The content endpoint returns the document plus its head version content.
func TestGetDocumentContent_OK(t *testing.T) {
	mock := &mockDocuments{head: &models.DocumentHead{
		Document: &models.Document{ID: "d1", Key: "a.md"},
		Head:     &models.Version{ID: "v3", VersionNumber: 3, Content: "# hello"},
	}}
	w := docDriver(t, mock).Request(t, http.MethodGet, "/documents/d1/content", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Document struct{ ID string } `json:"document"`
		Content  *struct {
			VersionID     string `json:"version_id"`
			VersionNumber int    `json:"version_number"`
			Content       string `json:"content"`
		} `json:"content"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Document.ID != "d1" || resp.Content == nil || resp.Content.VersionID != "v3" || resp.Content.Content != "# hello" {
		t.Errorf("content response = %s", w.Body.String())
	}
}

// An empty document (no versions) returns a null content block, not a 404.
func TestGetDocumentContent_EmptyDocument(t *testing.T) {
	mock := &mockDocuments{head: &models.DocumentHead{
		Document: &models.Document{ID: "d1", Key: "empty.md"},
		Head:     nil,
	}}
	w := docDriver(t, mock).Request(t, http.MethodGet, "/documents/d1/content", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Content *json.RawMessage `json:"content"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Content != nil && string(*resp.Content) != "null" {
		t.Errorf("empty-document content = %s, want null", w.Body.String())
	}
}

func TestDeleteDocument_PublishedConflict(t *testing.T) {
	w := docDriver(t, &mockDocuments{err: stores.ErrDocumentPublished}).Request(t, http.MethodDelete, "/documents/x", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateDocument_DuplicateConflict(t *testing.T) {
	mock := &mockDocuments{err: &pq.Error{Code: "23505"}}
	w := docDriver(t, mock).Request(t, http.MethodPost, "/documents", map[string]string{"key": "dup.md"})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestListDocuments_OK(t *testing.T) {
	pv := "v2"
	mock := &mockDocuments{list: []*models.Document{
		{ID: "d1", Key: "a.md", PublishedVersionID: &pv},
		{ID: "d2", Key: "b.md"},
	}}
	w := docDriver(t, mock).Request(t, http.MethodGet, "/documents?limit=10&offset=0", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total     int `json:"total"`
		Documents []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"documents"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	if resp.Documents[0].Status != "published" || resp.Documents[1].Status != "draft" {
		t.Errorf("list statuses = %q,%q; want published,draft", resp.Documents[0].Status, resp.Documents[1].Status)
	}
}

func TestListDocuments_ByTag(t *testing.T) {
	mock := &mockDocuments{list: []*models.Document{
		{ID: "d1", Key: "a.md"},
	}}
	w := docDriver(t, mock).Request(t, http.MethodGet, "/documents?tag=guide", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotTag != "guide" {
		t.Errorf("store got tag %q, want guide (tag filter routed to ListByTag)", mock.gotTag)
	}
}

func TestDeleteDocument_OK(t *testing.T) {
	mock := &mockDocuments{}
	w := docDriver(t, mock).Request(t, http.MethodDelete, "/documents/d1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if !mock.deleted {
		t.Error("delete was not called")
	}
}
