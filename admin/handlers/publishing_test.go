//go:build testing

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/admin/contracts"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockPublishing is a contracts.Publishing whose behavior each test sets.
type mockPublishing struct {
	doc        *models.Document
	err        error
	gotVersion string
}

func (m *mockPublishing) Publish(_ context.Context, _, versionID string) (*models.Document, error) {
	m.gotVersion = versionID
	return m.doc, m.err
}
func (m *mockPublishing) Unpublish(context.Context, string) (*models.Document, error) {
	return m.doc, m.err
}
func (m *mockPublishing) Rollback(_ context.Context, _, versionID string) (*models.Document, error) {
	m.gotVersion = versionID
	return m.doc, m.err
}

func pdriver(t *testing.T, mock contracts.Publishing) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Publishing](k, mock)
	}, All()...)
}

func TestPublishDocument_OK(t *testing.T) {
	pv := "v1"
	mock := &mockPublishing{doc: &models.Document{ID: "d1", PublishedVersionID: &pv}}
	w := pdriver(t, mock).Request(t, http.MethodPost, "/documents/d1/publish",
		map[string]string{"version_id": "v1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotVersion != "v1" {
		t.Errorf("store got version %q", mock.gotVersion)
	}
	var resp struct {
		PublishedVersionID *string `json:"published_version_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.PublishedVersionID == nil || *resp.PublishedVersionID != "v1" {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

func TestPublishDocument_VersionMismatch(t *testing.T) {
	w := pdriver(t, &mockPublishing{err: stores.ErrVersionMismatch}).Request(t, http.MethodPost,
		"/documents/d1/publish", map[string]string{"version_id": "foreign"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestPublishDocument_NotFound(t *testing.T) {
	w := pdriver(t, &mockPublishing{err: stores.ErrNotFound}).Request(t, http.MethodPost,
		"/documents/missing/publish", map[string]string{"version_id": "v1"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestUnpublishDocument_OK(t *testing.T) {
	mock := &mockPublishing{doc: &models.Document{ID: "d1"}}
	w := pdriver(t, mock).Request(t, http.MethodPost, "/documents/d1/unpublish", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRollbackDocument_OK(t *testing.T) {
	pv := "v1"
	mock := &mockPublishing{doc: &models.Document{ID: "d1", PublishedVersionID: &pv}}
	w := pdriver(t, mock).Request(t, http.MethodPost, "/documents/d1/rollback",
		map[string]string{"version_id": "v1"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotVersion != "v1" {
		t.Errorf("store got version %q", mock.gotVersion)
	}
}
