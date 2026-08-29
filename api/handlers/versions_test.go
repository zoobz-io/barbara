//go:build testing

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockVersions is a contracts.Versions whose behavior each test sets.
type mockVersions struct {
	v          *models.Version
	list       []*models.Version
	err        error
	gotDoc     string
	gotContent string
	gotBase    int
}

func (m *mockVersions) Save(_ context.Context, documentID, content string, baseVersion int) (*models.Version, error) {
	m.gotDoc, m.gotContent, m.gotBase = documentID, content, baseVersion
	return m.v, m.err
}
func (m *mockVersions) List(context.Context, string, int, int) ([]*models.Version, error) {
	return m.list, m.err
}
func (m *mockVersions) Get(context.Context, string) (*models.Version, error) {
	return m.v, m.err
}

func vdriver(t *testing.T, mock contracts.Versions) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Versions](k, mock)
	}, All()...)
}

func TestSaveVersion_OK(t *testing.T) {
	mock := &mockVersions{v: &models.Version{ID: "v1", DocumentID: "d1", VersionNumber: 3}}
	w := vdriver(t, mock).Request(t, http.MethodPost, "/documents/d1/versions",
		map[string]any{"content": "# hello", "base_version": 2})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotDoc != "d1" || mock.gotContent != "# hello" || mock.gotBase != 2 {
		t.Errorf("store got doc=%q content=%q base=%d", mock.gotDoc, mock.gotContent, mock.gotBase)
	}
	var resp struct {
		ID            string `json:"id"`
		VersionNumber int    `json:"version_number"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "v1" || resp.VersionNumber != 3 {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

// base_version is required — an omitted one is a 400 before the store is touched.
func TestSaveVersion_MissingBaseVersion(t *testing.T) {
	mock := &mockVersions{}
	w := vdriver(t, mock).Request(t, http.MethodPost, "/documents/d1/versions",
		map[string]any{"content": "# hello"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
	}
	if mock.gotContent != "" {
		t.Error("store was called despite a missing base_version")
	}
}

// A stale base_version is a 409 carrying the current head.
func TestSaveVersion_Conflict(t *testing.T) {
	mock := &mockVersions{err: &stores.VersionConflictError{CurrentHead: 6}}
	w := vdriver(t, mock).Request(t, http.MethodPost, "/documents/d1/versions",
		map[string]any{"content": "# hello", "base_version": 3})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "6") {
		t.Errorf("409 body should report the current head (6): %s", w.Body.String())
	}
}

func TestSaveVersion_DocumentNotFound(t *testing.T) {
	w := vdriver(t, &mockVersions{err: stores.ErrNotFound}).Request(t, http.MethodPost,
		"/documents/missing/versions", map[string]any{"content": "x", "base_version": 0})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestListVersions_OK(t *testing.T) {
	mock := &mockVersions{list: []*models.Version{
		{ID: "v2", VersionNumber: 2}, {ID: "v1", VersionNumber: 1},
	}}
	w := vdriver(t, mock).Request(t, http.MethodGet, "/documents/d1/versions", nil)
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

func TestGetVersion_NotFound(t *testing.T) {
	w := vdriver(t, &mockVersions{err: stores.ErrNotFound}).Request(t, http.MethodGet, "/versions/x", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}
