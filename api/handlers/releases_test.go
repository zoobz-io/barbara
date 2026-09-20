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
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockReleases is a contracts.Releases whose behavior each test sets.
type mockReleases struct {
	release  *models.Release
	list     []*models.Release
	entries  []*models.ReleaseEntry
	changes  []*models.ReleaseChange
	err      error
	total    int64
	gotApp   string
	gotID    string
	gotLabel string
}

func (m *mockReleases) Cut(_ context.Context, appID, label string) (*models.Release, error) {
	m.gotApp, m.gotLabel = appID, label
	return m.release, m.err
}
func (m *mockReleases) List(_ context.Context, appID string, _, _ int) ([]*models.Release, error) {
	m.gotApp = appID
	return m.list, m.err
}
func (m *mockReleases) Total(_ context.Context, appID string) (int64, error) {
	m.gotApp = appID
	return m.total, m.err
}
func (m *mockReleases) Get(_ context.Context, appID, id string) (*models.Release, []*models.ReleaseEntry, error) {
	m.gotApp, m.gotID = appID, id
	return m.release, m.entries, m.err
}
func (m *mockReleases) Changes(_ context.Context, appID, id string) (*models.Release, []*models.ReleaseChange, error) {
	m.gotApp, m.gotID = appID, id
	return m.release, m.changes, m.err
}
func (m *mockReleases) Rollback(_ context.Context, appID, id, label string) (*models.Release, error) {
	m.gotApp, m.gotID, m.gotLabel = appID, id, label
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

// The optional body carries a label, handed to the store as given.
func TestCutRelease_WithLabel(t *testing.T) {
	label := "Launch"
	mock := &mockReleases{release: &models.Release{ID: "r-1", Number: 1, Kind: models.ReleaseKindCut, Label: &label}}
	w := rdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/releases", wire.CutReleaseRequest{Label: "Launch"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotLabel != "Launch" {
		t.Errorf("store got label = %q, want Launch", mock.gotLabel)
	}
	var resp struct {
		Kind  string `json:"kind"`
		Label string `json:"label"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Kind != "cut" || resp.Label != "Launch" {
		t.Errorf("unexpected response: %s", w.Body.String())
	}
}

// A label past the bound is rejected before the store is reached.
func TestCutRelease_LabelTooLong(t *testing.T) {
	mock := &mockReleases{release: &models.Release{ID: "r-1"}}
	long := strings.Repeat("x", wire.MaxReleaseLabelLen+1)
	w := rdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/releases", wire.CutReleaseRequest{Label: long})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
	}
	if mock.gotApp != "" {
		t.Error("store was called for an invalid label")
	}
}

func TestCutRelease_AppNotFound(t *testing.T) {
	w := rdriver(t, &mockReleases{err: stores.ErrNotFound}).Request(t, http.MethodPost, "/apps/missing/releases", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// The list carries the app's true total, not the page size, and each row's
// at-a-glance fields.
func TestListReleases_OK(t *testing.T) {
	mock := &mockReleases{
		list:  []*models.Release{{ID: "r-2", Number: 2, Kind: models.ReleaseKindPublish, EntryCount: 5, Changed: 1}, {ID: "r-1", Number: 1}},
		total: 12,
	}
	w := rdriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/releases?limit=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Releases []struct {
			Kind       string `json:"kind"`
			EntryCount int    `json:"entry_count"`
			Changed    int    `json:"changed"`
		} `json:"releases"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 12 {
		t.Errorf("total = %d, want 12 (the app's count, not the page's)", resp.Total)
	}
	if len(resp.Releases) != 2 || resp.Releases[0].Kind != "publish" || resp.Releases[0].EntryCount != 5 || resp.Releases[0].Changed != 1 {
		t.Errorf("list response = %s", w.Body.String())
	}
}

// The changes endpoint returns the release with its diff rows.
func TestGetReleaseChanges_OK(t *testing.T) {
	prevKey, prevN, newN := "old.md", 1, 2
	mock := &mockReleases{
		release: &models.Release{ID: "r-2", Number: 2, Moved: 1},
		changes: []*models.ReleaseChange{{Key: "new.md", DocumentID: "d-1", Change: models.ChangeMoved,
			PrevKey: &prevKey, PrevVersionNumber: &prevN, VersionNumber: &newN}},
	}
	w := rdriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/releases/r-2/changes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "r-2" {
		t.Errorf("store got id = %q", mock.gotID)
	}
	var resp struct {
		Release struct {
			Moved int `json:"moved"`
		} `json:"release"`
		Changes []struct {
			Key        string `json:"key"`
			Change     string `json:"change"`
			PrevKey    string `json:"prev_key"`
			PrevNumber int    `json:"prev_version_number"`
			Number     int    `json:"version_number"`
		} `json:"changes"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Release.Moved != 1 || len(resp.Changes) != 1 || resp.Changes[0].Change != "moved" ||
		resp.Changes[0].PrevKey != "old.md" || resp.Changes[0].PrevNumber != 1 || resp.Changes[0].Number != 2 {
		t.Errorf("changes response = %s", w.Body.String())
	}
}

func TestGetReleaseChanges_NotFound(t *testing.T) {
	w := rdriver(t, &mockReleases{err: stores.ErrNotFound}).Request(t, http.MethodGet, "/apps/app-1/releases/r-x/changes", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
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
	src := "r-3"
	mock := &mockReleases{release: &models.Release{ID: "r-9", Number: 9, Kind: models.ReleaseKindRollback, SourceReleaseID: &src}}
	w := rdriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/releases/r-3/rollback", wire.CutReleaseRequest{Label: "back"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.gotID != "r-3" || mock.gotLabel != "back" {
		t.Errorf("store got id = %q, label = %q", mock.gotID, mock.gotLabel)
	}
	var resp struct {
		Number int    `json:"number"`
		Kind   string `json:"kind"`
		Source string `json:"source_release_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Number != 9 || resp.Kind != "rollback" || resp.Source != "r-3" {
		t.Errorf("rollback response = %s, want the new forward release (9) sourced from r-3", w.Body.String())
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
