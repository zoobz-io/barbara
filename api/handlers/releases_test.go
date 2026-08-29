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

// mockReleases is a contracts.Releases whose behavior each test sets.
type mockReleases struct {
	release *models.Release
	list    []*models.Release
	entries []*models.ReleaseEntry
	err     error
	gotApp  string
	gotID   string
}

func (m *mockReleases) Cut(_ context.Context, appID string) (*models.Release, error) {
	m.gotApp = appID
	return m.release, m.err
}
func (m *mockReleases) List(_ context.Context, appID string, _, _ int) ([]*models.Release, error) {
	m.gotApp = appID
	return m.list, m.err
}
func (m *mockReleases) Get(_ context.Context, appID, id string) (*models.Release, []*models.ReleaseEntry, error) {
	m.gotApp, m.gotID = appID, id
	return m.release, m.entries, m.err
}
func (m *mockReleases) Rollback(_ context.Context, appID, id string) (*models.Release, error) {
	m.gotApp, m.gotID = appID, id
	return m.release, m.err
}

func rdriver(t *testing.T, mock contracts.Releases) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Releases](k, mock)
	}, All()...)
}

func TestCutRelease_OK(t *testing.T) {
	mock := &mockReleases{release: &models.Release{ID: "r-1", AppID: "app-1", Number: 3}}
	w := rdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/releases", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "app-1" {
		t.Errorf("store got app = %q", mock.gotApp)
	}
	var resp struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "r-1" || resp.Number != 3 {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

func TestCutRelease_AppNotFound(t *testing.T) {
	w := rdriver(t, &mockReleases{err: stores.ErrNotFound}).Request(t, http.MethodPost, "/apps/missing/releases", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestListReleases_OK(t *testing.T) {
	mock := &mockReleases{list: []*models.Release{{ID: "r-2", Number: 2}, {ID: "r-1", Number: 1}}}
	w := rdriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/releases", nil)
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

func TestGetRelease_WithEntries(t *testing.T) {
	mock := &mockReleases{
		release: &models.Release{ID: "r-1", Number: 1},
		entries: []*models.ReleaseEntry{{Key: "a.md", DocumentID: "d-1", VersionID: "v-1"}},
	}
	w := rdriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/releases/r-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "r-1" {
		t.Errorf("store got id = %q", mock.gotID)
	}
	var resp struct {
		Release struct {
			ID string `json:"id"`
		} `json:"release"`
		Entries []struct {
			Key string `json:"key"`
		} `json:"entries"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Release.ID != "r-1" || len(resp.Entries) != 1 || resp.Entries[0].Key != "a.md" {
		t.Errorf("get response = %s", w.Body.String())
	}
}

func TestRollbackRelease_OK(t *testing.T) {
	mock := &mockReleases{release: &models.Release{ID: "r-9", Number: 9}}
	w := rdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/releases/r-3/rollback", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "r-3" {
		t.Errorf("store got id = %q", mock.gotID)
	}
	var resp struct {
		Number int `json:"number"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Number != 9 {
		t.Errorf("rollback returned number %d, want the new forward release (9)", resp.Number)
	}
}

// Cutting is a publish operation: read+write scopes are not enough.
func TestCutRelease_RequiresPublishScope(t *testing.T) {
	noPublish := auth.NewStub("u", "t", "", nil, []string{auth.ScopeDocumentsRead, auth.ScopeDocumentsWrite})
	d := testkit.HandlersAs(t, noPublish, func(k sum.Key) {
		sum.Register[contracts.Releases](k, &mockReleases{})
	}, All()...)

	w := d.Request(t, http.MethodPost, "/apps/app-1/releases", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cut without publish scope = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}
