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

// mockVersions is a contracts.Versions whose behavior each test sets.
type mockVersions struct {
	v          *models.Version
	list       []*models.Version
	err        error
	gotDoc     string
	gotContent string
}

func (m *mockVersions) Save(_ context.Context, documentID, content string) (*models.Version, error) {
	m.gotDoc, m.gotContent = documentID, content
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
		map[string]string{"content": "# hello"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotDoc != "d1" || mock.gotContent != "# hello" {
		t.Errorf("store got doc=%q content=%q", mock.gotDoc, mock.gotContent)
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

func TestSaveVersion_DocumentNotFound(t *testing.T) {
	w := vdriver(t, &mockVersions{err: stores.ErrNotFound}).Request(t, http.MethodPost,
		"/documents/missing/versions", map[string]string{"content": "x"})
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
