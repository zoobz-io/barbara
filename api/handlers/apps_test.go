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
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockApps is a contracts.Apps whose behavior each test sets.
type mockApps struct {
	app     *models.App
	list    []*models.App
	err     error
	gotName string
	gotID   string
}

func (m *mockApps) Create(_ context.Context, name string) (*models.App, error) {
	m.gotName = name
	return m.app, m.err
}
func (m *mockApps) Get(_ context.Context, id string) (*models.App, error) {
	m.gotID = id
	return m.app, m.err
}
func (m *mockApps) List(context.Context, int, int) ([]*models.App, error) {
	return m.list, m.err
}
func (m *mockApps) Rename(_ context.Context, id, newName string) (*models.App, error) {
	m.gotID, m.gotName = id, newName
	return m.app, m.err
}
func (m *mockApps) Delete(_ context.Context, id string) error {
	m.gotID = id
	return m.err
}

func adriver(t *testing.T, mock contracts.Apps) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Apps](k, mock)
	}, All()...)
}

func TestCreateApp_OK(t *testing.T) {
	mock := &mockApps{app: &models.App{ID: "app-1", TenantID: "t", Name: "docs-site"}}
	w := adriver(t, mock).Request(t, http.MethodPost, "/apps",
		map[string]any{"name": "docs-site"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotName != "docs-site" {
		t.Errorf("store got name = %q", mock.gotName)
	}
	var resp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "app-1" || resp.Name != "docs-site" {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

// An empty name is a 422 before the store is touched.
func TestCreateApp_MissingName(t *testing.T) {
	mock := &mockApps{}
	w := adriver(t, mock).Request(t, http.MethodPost, "/apps", map[string]any{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
	}
	if mock.gotName != "" {
		t.Error("store was called despite a missing name")
	}
}

// A duplicate name is a 409.
func TestCreateApp_NameTaken(t *testing.T) {
	mock := &mockApps{err: stores.ErrAppNameTaken}
	w := adriver(t, mock).Request(t, http.MethodPost, "/apps",
		map[string]any{"name": "docs-site"})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestGetApp_OK(t *testing.T) {
	mock := &mockApps{app: &models.App{ID: "app-1", Name: "docs-site"}}
	w := adriver(t, mock).Request(t, http.MethodGet, "/apps/app-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "app-1" {
		t.Errorf("store got id = %q", mock.gotID)
	}
}

func TestGetApp_NotFound(t *testing.T) {
	w := adriver(t, &mockApps{err: stores.ErrNotFound}).Request(t, http.MethodGet, "/apps/missing", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestListApps_OK(t *testing.T) {
	mock := &mockApps{list: []*models.App{
		{ID: "app-2", Name: "b"}, {ID: "app-1", Name: "a"},
	}}
	w := adriver(t, mock).Request(t, http.MethodGet, "/apps", nil)
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

func TestRenameApp_OK(t *testing.T) {
	mock := &mockApps{app: &models.App{ID: "app-1", Name: "marketing-site"}}
	w := adriver(t, mock).Request(t, http.MethodPatch, "/apps/app-1",
		map[string]any{"name": "marketing-site"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "app-1" || mock.gotName != "marketing-site" {
		t.Errorf("store got id=%q name=%q", mock.gotID, mock.gotName)
	}
}

func TestDeleteApp_OK(t *testing.T) {
	mock := &mockApps{}
	w := adriver(t, mock).Request(t, http.MethodDelete, "/apps/app-1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "app-1" {
		t.Errorf("store got id = %q", mock.gotID)
	}
}

// Deleting an app that has releases is a 409 carrying the reason.
func TestDeleteApp_HasReleases(t *testing.T) {
	mock := &mockApps{err: stores.ErrAppHasReleases}
	w := adriver(t, mock).Request(t, http.MethodDelete, "/apps/app-1", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "releases") {
		t.Errorf("409 body should explain the release guard: %s", w.Body.String())
	}
}

// Create is a write operation: a read-only identity is refused with 403.
func TestApps_ScopeGate_WriteRequired(t *testing.T) {
	reader := auth.NewStub("u", "t", "", nil, []string{auth.ScopeDocumentsRead})
	d := testkit.HandlersAs(t, reader, func(k sum.Key) {
		sum.Register[contracts.Apps](k, &mockApps{})
	}, All()...)

	w := d.Request(t, http.MethodPost, "/apps", map[string]any{"name": "docs-site"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("create with read-only scope = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}
