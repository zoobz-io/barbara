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

// mockCollections is a contracts.Collections whose behavior each test sets.
type mockCollections struct {
	col         *models.Collection
	contents    *models.CollectionContents
	err         error
	gotApp      string
	gotID       string
	gotName     string
	gotParent   *string
	gotListedID *string
}

func (m *mockCollections) Create(_ context.Context, appID string, parentID *string, name string) (*models.Collection, error) {
	m.gotApp, m.gotParent, m.gotName = appID, parentID, name
	return m.col, m.err
}
func (m *mockCollections) Get(_ context.Context, appID, id string) (*models.Collection, error) {
	m.gotApp, m.gotID = appID, id
	return m.col, m.err
}
func (m *mockCollections) ListContents(_ context.Context, appID string, collectionID *string) (*models.CollectionContents, error) {
	m.gotApp, m.gotListedID = appID, collectionID
	return m.contents, m.err
}
func (m *mockCollections) Rename(_ context.Context, appID, id, newName string) (*models.Collection, error) {
	m.gotApp, m.gotID, m.gotName = appID, id, newName
	return m.col, m.err
}
func (m *mockCollections) Move(_ context.Context, appID, id string, newParentID *string) (*models.Collection, error) {
	m.gotApp, m.gotID, m.gotParent = appID, id, newParentID
	return m.col, m.err
}
func (m *mockCollections) Delete(_ context.Context, appID, id string) error {
	m.gotApp, m.gotID = appID, id
	return m.err
}

func cdriver(t *testing.T, mock contracts.Collections) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Collections](k, mock)
	}, All()...)
}

func TestCreateCollection_OK(t *testing.T) {
	mock := &mockCollections{col: &models.Collection{ID: "c-1", AppID: "app-1", Name: "guides"}}
	w := cdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/collections",
		map[string]any{"name": "guides"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "app-1" || mock.gotName != "guides" || mock.gotParent != nil {
		t.Errorf("store got app=%q name=%q parent=%v", mock.gotApp, mock.gotName, mock.gotParent)
	}
}

func TestCreateCollection_UnderParent(t *testing.T) {
	mock := &mockCollections{col: &models.Collection{ID: "c-2"}}
	w := cdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/collections",
		map[string]any{"name": "sub", "parent_id": "c-1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotParent == nil || *mock.gotParent != "c-1" {
		t.Errorf("store got parent = %v, want c-1", mock.gotParent)
	}
}

func TestCreateCollection_MissingName(t *testing.T) {
	mock := &mockCollections{}
	w := cdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/collections", map[string]any{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
	}
	if mock.gotName != "" {
		t.Error("store was called despite a missing name")
	}
}

func TestCreateCollection_NameTaken(t *testing.T) {
	mock := &mockCollections{err: stores.ErrCollectionNameTaken}
	w := cdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/collections",
		map[string]any{"name": "guides"})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestGetCollection_NotFound(t *testing.T) {
	w := cdriver(t, &mockCollections{err: stores.ErrNotFound}).Request(t, http.MethodGet,
		"/apps/app-1/collections/missing", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestListAppRootContents_OK(t *testing.T) {
	mock := &mockCollections{contents: &models.CollectionContents{
		Subcollections: []*models.Collection{{ID: "c-1", Name: "guides"}},
		Documents:      []*models.Document{{ID: "d-1", Key: "readme.md"}},
	}}
	w := cdriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/contents", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotListedID != nil {
		t.Errorf("root listing passed a collection id: %v", mock.gotListedID)
	}
	var resp struct {
		Subcollections []json.RawMessage `json:"subcollections"`
		Documents      []json.RawMessage `json:"documents"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Subcollections) != 1 || len(resp.Documents) != 1 {
		t.Errorf("contents = %s", w.Body.String())
	}
}

func TestListCollectionContents_OK(t *testing.T) {
	mock := &mockCollections{contents: &models.CollectionContents{}}
	w := cdriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/collections/c-1/contents", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotListedID == nil || *mock.gotListedID != "c-1" {
		t.Errorf("collection listing passed id = %v, want c-1", mock.gotListedID)
	}
}

func TestRenameCollection_OK(t *testing.T) {
	mock := &mockCollections{col: &models.Collection{ID: "c-1", Name: "tutorials"}}
	w := cdriver(t, mock).Request(t, http.MethodPatch, "/apps/app-1/collections/c-1",
		map[string]any{"name": "tutorials"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "c-1" || mock.gotName != "tutorials" {
		t.Errorf("store got id=%q name=%q", mock.gotID, mock.gotName)
	}
}

func TestMoveCollection_ToRoot(t *testing.T) {
	mock := &mockCollections{col: &models.Collection{ID: "c-1"}}
	w := cdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/collections/c-1/move",
		map[string]any{"parent_id": nil})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "c-1" || mock.gotParent != nil {
		t.Errorf("store got id=%q parent=%v, want c-1/nil", mock.gotID, mock.gotParent)
	}
}

func TestMoveCollection_Cycle(t *testing.T) {
	mock := &mockCollections{err: stores.ErrCollectionCycle}
	w := cdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/collections/c-1/move",
		map[string]any{"parent_id": "d-1"})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "descendant") {
		t.Errorf("409 body should explain the cycle: %s", w.Body.String())
	}
}

func TestDeleteCollection_OK(t *testing.T) {
	mock := &mockCollections{}
	w := cdriver(t, mock).Request(t, http.MethodDelete, "/apps/app-1/collections/c-1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "c-1" {
		t.Errorf("store got id = %q", mock.gotID)
	}
}

func TestDeleteCollection_NotEmpty(t *testing.T) {
	mock := &mockCollections{err: stores.ErrCollectionNotEmpty}
	w := cdriver(t, mock).Request(t, http.MethodDelete, "/apps/app-1/collections/c-1", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not empty") {
		t.Errorf("409 body should explain the non-empty guard: %s", w.Body.String())
	}
}

// Create is a write op: a read-only identity is refused with 403.
func TestCollections_ScopeGate_WriteRequired(t *testing.T) {
	reader := auth.NewStub("u", "t", "", nil, []string{auth.ScopeDocumentsRead})
	d := testkit.HandlersAs(t, reader, func(k sum.Key) {
		sum.Register[contracts.Collections](k, &mockCollections{})
	}, All()...)

	w := d.Request(t, http.MethodPost, "/apps/app-1/collections", map[string]any{"name": "guides"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("create with read-only scope = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}
