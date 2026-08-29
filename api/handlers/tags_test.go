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
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockTagging is a contracts.Tagging whose behavior each test sets.
type mockTagging struct {
	doc    *models.Document
	err    error
	gotID  string
	gotTag string
}

func (m *mockTagging) AddTag(_ context.Context, id, tag string) (*models.Document, error) {
	m.gotID, m.gotTag = id, tag
	return m.doc, m.err
}
func (m *mockTagging) RemoveTag(_ context.Context, id, tag string) (*models.Document, error) {
	m.gotID, m.gotTag = id, tag
	return m.doc, m.err
}

// tagDriver wires the tag handlers over the mock contract.
func tagDriver(t *testing.T, mock contracts.Tagging) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Tagging](k, mock)
		sum.Register[contracts.Documents](k, &mockDocuments{status: "published"})
	}, All()...)
}

func TestAddDocumentTag_OK(t *testing.T) {
	mock := &mockTagging{doc: &models.Document{ID: "d1", Key: "a.md", Tags: []string{"guide"}}}
	w := tagDriver(t, mock).Request(t, http.MethodPost, "/documents/d1/tags", map[string]string{"tag": "guide"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "d1" || mock.gotTag != "guide" {
		t.Errorf("store got id=%q tag=%q, want d1/guide", mock.gotID, mock.gotTag)
	}
	var resp struct {
		Tags []string `json:"tags"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Tags) != 1 || resp.Tags[0] != "guide" {
		t.Errorf("response tags = %v, want [guide]", resp.Tags)
	}
}

func TestAddDocumentTag_MissingTag(t *testing.T) {
	mock := &mockTagging{}
	w := tagDriver(t, mock).Request(t, http.MethodPost, "/documents/d1/tags", map[string]string{"tag": ""})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if mock.gotTag != "" || mock.gotID != "" {
		t.Error("store was called despite an empty tag")
	}
}

func TestAddDocumentTag_NotFound(t *testing.T) {
	w := tagDriver(t, &mockTagging{err: stores.ErrNotFound}).
		Request(t, http.MethodPost, "/documents/x/tags", map[string]string{"tag": "guide"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveDocumentTag_OK(t *testing.T) {
	mock := &mockTagging{doc: &models.Document{ID: "d1", Key: "a.md"}}
	w := tagDriver(t, mock).Request(t, http.MethodDelete, "/documents/d1/tags?tag=guide", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "d1" || mock.gotTag != "guide" {
		t.Errorf("store got id=%q tag=%q, want d1/guide", mock.gotID, mock.gotTag)
	}
}

func TestRemoveDocumentTag_MissingTag(t *testing.T) {
	mock := &mockTagging{}
	w := tagDriver(t, mock).Request(t, http.MethodDelete, "/documents/d1/tags", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if mock.gotTag != "" {
		t.Error("store was called despite a missing tag param")
	}
}
