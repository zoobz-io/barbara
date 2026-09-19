//go:build testing

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockAssets is a contracts.Assets whose behavior each test sets.
type mockAssets struct {
	asset      *models.Asset
	list       []*models.Asset
	err        error
	putApp     string
	putKey     string
	putCT      string
	putData    []byte
	getApp     string
	getKey     string
	listApp    string
	listPrefix string
	level      *models.AssetLevel
	levelApp   string
	levelPath  string
	created    string
	stats      *models.AssetStats
	statsApp   string
	moveKey    string
	moveTo     string
	delApp     string
	delKey     string
}

func (m *mockAssets) Put(_ context.Context, appID, key, contentType string, data []byte) (*models.Asset, error) {
	m.putApp, m.putKey, m.putCT, m.putData = appID, key, contentType, data
	if m.err != nil {
		return nil, m.err
	}
	if m.asset != nil {
		return m.asset, nil
	}
	return &models.Asset{Key: key, ContentType: contentType, Size: int64(len(data))}, nil
}

func (m *mockAssets) Get(_ context.Context, appID, key string) (*models.Asset, error) {
	m.getApp, m.getKey = appID, key
	return m.asset, m.err
}

func (m *mockAssets) List(_ context.Context, appID, keyPrefix string) ([]*models.Asset, error) {
	m.listApp, m.listPrefix = appID, keyPrefix
	return m.list, m.err
}

func (m *mockAssets) ListFolder(_ context.Context, appID, folderPath string) (*models.AssetLevel, error) {
	m.levelApp, m.levelPath = appID, folderPath
	return m.level, m.err
}

func (m *mockAssets) CreateFolder(_ context.Context, appID, folderPath string) (*models.AssetLevel, error) {
	m.levelApp, m.created = appID, folderPath
	return m.level, m.err
}

func (m *mockAssets) Stats(_ context.Context, appID string) (*models.AssetStats, error) {
	m.statsApp = appID
	return m.stats, m.err
}

func (m *mockAssets) Move(_ context.Context, appID, key, newKey string) (*models.Asset, error) {
	m.levelApp, m.moveKey, m.moveTo = appID, key, newKey
	return m.asset, m.err
}

func (m *mockAssets) Delete(_ context.Context, appID, key string) error {
	m.delApp, m.delKey = appID, key
	return m.err
}

func assetDriver(t *testing.T, mock contracts.Assets) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Assets](k, mock)
	}, All()...)
}

// Upload passes the app, the raw body, and its content type through to the
// store and echoes back the stored metadata.
func TestUploadAsset_OK(t *testing.T) {
	mock := &mockAssets{}
	w := assetDriver(t, mock).RequestRaw(t, testkit.DefaultTenant,
		http.MethodPut, "/apps/app-1/assets/object?key=images/logo.png", "image/png", []byte("PNGBYTES"))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.putApp != "app-1" || mock.putKey != "images/logo.png" || mock.putCT != "image/png" || !bytes.Equal(mock.putData, []byte("PNGBYTES")) {
		t.Errorf("store got app=%q key=%q ct=%q data=%q", mock.putApp, mock.putKey, mock.putCT, mock.putData)
	}
	var resp struct {
		Key         string `json:"key"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Key != "images/logo.png" || resp.ContentType != "image/png" || resp.Size != 8 {
		t.Errorf("response = %+v, want the stored metadata", resp)
	}
}

// Download writes the raw bytes with the stored content type, bypassing JSON.
func TestGetAsset_OK(t *testing.T) {
	mock := &mockAssets{asset: &models.Asset{Key: "doc.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")}}
	w := assetDriver(t, mock).RequestRaw(t, testkit.DefaultTenant,
		http.MethodGet, "/apps/app-1/assets/object?key=doc.pdf", "", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.getApp != "app-1" || mock.getKey != "doc.pdf" {
		t.Errorf("store got app=%q key=%q, want app-1/doc.pdf", mock.getApp, mock.getKey)
	}
	if !bytes.Equal(w.Body.Bytes(), []byte("%PDF-1.7")) {
		t.Errorf("body = %q, want the raw asset bytes", w.Body.Bytes())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
}

func TestGetAsset_NotFound(t *testing.T) {
	w := assetDriver(t, &mockAssets{err: stores.ErrNotFound}).RequestRaw(t, testkit.DefaultTenant,
		http.MethodGet, "/apps/app-1/assets/object?key=missing", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// List is app-scoped and passes the optional prefix (folder view) through.
func TestListAssets_OK(t *testing.T) {
	mock := &mockAssets{list: []*models.Asset{
		{Key: "images/a.png", ContentType: "image/png", Size: 10},
		{Key: "images/b.png", ContentType: "image/png", Size: 20},
	}}
	w := assetDriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/assets?prefix=images/", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.listApp != "app-1" || mock.listPrefix != "images/" {
		t.Errorf("store got app=%q prefix=%q, want app-1/images/", mock.listApp, mock.listPrefix)
	}
	var resp struct {
		Assets []struct {
			Key  string `json:"key"`
			Kind string `json:"kind"`
			Size int64  `json:"size"`
		} `json:"assets"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 || len(resp.Assets) != 2 || resp.Assets[0].Key != "images/a.png" || resp.Assets[0].Kind != "image" {
		t.Errorf("list response = %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "last_modified") {
		t.Errorf("list response carries a timestamp the bucket never reported: %s", w.Body.String())
	}
}

func TestListAssetFolder_OK(t *testing.T) {
	written := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	mock := &mockAssets{level: &models.AssetLevel{
		Path:    "images",
		Folders: []models.AssetFolder{{Name: "icons", Count: 2, Size: 40, LastWrittenAt: &written}},
		Assets:  []*models.Asset{{Key: "images/logo.png", ContentType: "image/png", Size: 10, LastModified: written}},
	}}
	w := assetDriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/assets/folder?path=images", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.levelApp != "app-1" || mock.levelPath != "images" {
		t.Errorf("store got app=%q path=%q, want app-1/images", mock.levelApp, mock.levelPath)
	}
	var resp struct {
		Path    string `json:"path"`
		Folders []struct {
			Name          string  `json:"name"`
			Count         int     `json:"count"`
			Size          int64   `json:"size"`
			LastWrittenAt *string `json:"last_written_at"`
		} `json:"folders"`
		Assets []struct {
			Key          string  `json:"key"`
			LastModified *string `json:"last_modified"`
		} `json:"assets"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Path != "images" || len(resp.Folders) != 1 || resp.Folders[0].Name != "icons" || resp.Folders[0].Count != 2 ||
		resp.Folders[0].Size != 40 || len(resp.Assets) != 1 || resp.Assets[0].Key != "images/logo.png" {
		t.Errorf("folder response = %s", w.Body.String())
	}
	if resp.Folders[0].LastWrittenAt == nil || resp.Assets[0].LastModified == nil {
		t.Errorf("folder response omits the timestamps: %s", w.Body.String())
	}
}

// An empty level serializes with empty arrays, so the browser can render it
// without null checks.
func TestListAssetFolder_Empty(t *testing.T) {
	mock := &mockAssets{level: &models.AssetLevel{Folders: []models.AssetFolder{}, Assets: []*models.Asset{}}}
	w := assetDriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/assets/folder", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.levelPath != "" {
		t.Errorf("store got path=%q, want empty for the root", mock.levelPath)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"folders":[]`) || !strings.Contains(body, `"assets":[]`) {
		t.Errorf("empty level = %s, want empty arrays", body)
	}
}

// Creating a folder passes the body's path to the store and answers 201 with
// the folder's level.
func TestCreateAssetFolder_OK(t *testing.T) {
	mock := &mockAssets{level: &models.AssetLevel{Path: "images/icons", Folders: []models.AssetFolder{}, Assets: []*models.Asset{}}}
	w := assetDriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/assets/folder",
		map[string]any{"path": "images/icons"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if mock.levelApp != "app-1" || mock.created != "images/icons" {
		t.Errorf("store got app=%q path=%q, want app-1/images/icons", mock.levelApp, mock.created)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"path":"images/icons"`) || !strings.Contains(body, `"folders":[]`) {
		t.Errorf("create response = %s, want the empty level at images/icons", body)
	}
}

// A path the store rejects is a 400.
func TestCreateAssetFolder_Invalid(t *testing.T) {
	mock := &mockAssets{err: stores.ErrInvalidAssetPath}
	w := assetDriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/assets/folder",
		map[string]any{"path": "a//b"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

// Stats maps the bookkeeping view: totals from the root row, kinds as given,
// and the flat day rows regrouped per day with the all-kinds row as the
// day's total.
func TestGetAssetStats_OK(t *testing.T) {
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	written := day.Add(9 * time.Hour)
	mock := &mockAssets{stats: &models.AssetStats{
		ComputedAt: &written,
		Root:       &models.AssetFolderStat{Count: 3, Bytes: 300, LastWrittenAt: &written},
		Kinds: []*models.AssetKindStat{
			{Kind: models.KindImage, Count: 2, Bytes: 200, LastWrittenAt: &written},
			{Kind: models.KindText, Count: 1, Bytes: 100},
		},
		Days: []*models.AssetDayStat{
			{Day: day, Kind: "", Count: 3, Bytes: 300},
			{Day: day, Kind: models.KindImage, Count: 2, Bytes: 200},
			{Day: day, Kind: models.KindText, Count: 1, Bytes: 100},
		},
	}}
	w := assetDriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/assets/stats", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.statsApp != "app-1" {
		t.Errorf("store got app=%q, want app-1", mock.statsApp)
	}
	var resp struct {
		ComputedAt    *string `json:"computed_at"`
		LastWrittenAt *string `json:"last_written_at"`
		Count         int64   `json:"count"`
		Size          int64   `json:"size"`
		Kinds         []struct {
			Kind  string `json:"kind"`
			Count int64  `json:"count"`
		} `json:"kinds"`
		Days []struct {
			Day   string `json:"day"`
			Count int64  `json:"count"`
			Size  int64  `json:"size"`
			Kinds []struct {
				Kind  string `json:"kind"`
				Count int64  `json:"count"`
			} `json:"kinds"`
		} `json:"days"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 3 || resp.Size != 300 || resp.ComputedAt == nil || resp.LastWrittenAt == nil {
		t.Errorf("totals = %s", w.Body.String())
	}
	if len(resp.Kinds) != 2 || resp.Kinds[0].Kind != "image" || resp.Kinds[0].Count != 2 || resp.Kinds[1].Kind != "text" {
		t.Errorf("kinds = %s", w.Body.String())
	}
	if len(resp.Days) != 1 || resp.Days[0].Day != "2026-09-16" || resp.Days[0].Count != 3 || resp.Days[0].Size != 300 ||
		len(resp.Days[0].Kinds) != 2 || resp.Days[0].Kinds[0].Kind != "image" || resp.Days[0].Kinds[1].Count != 1 {
		t.Errorf("days = %s", w.Body.String())
	}
}

// An app with no assets still answers a full shape: zero totals, no
// timestamps, and empty arrays.
func TestGetAssetStats_Empty(t *testing.T) {
	mock := &mockAssets{stats: &models.AssetStats{Root: &models.AssetFolderStat{}}}
	w := assetDriver(t, mock).Request(t, http.MethodGet, "/apps/app-1/assets/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"count":0`, `"kinds":[]`, `"days":[]`} {
		if !strings.Contains(body, want) {
			t.Errorf("empty stats = %s, want %s", body, want)
		}
	}
	if strings.Contains(body, "computed_at") || strings.Contains(body, "last_written_at") {
		t.Errorf("empty stats = %s, want the timestamps omitted", body)
	}
}

// Moving passes the source key from the query and the destination from the
// body, and answers with the asset at its new key.
func TestMoveAsset_OK(t *testing.T) {
	mock := &mockAssets{asset: &models.Asset{Key: "images/brand/logo.png", ContentType: "image/png", Size: 10}}
	w := assetDriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/assets/object/move?key=images/logo.png",
		map[string]any{"key": "images/brand/logo.png"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.levelApp != "app-1" || mock.moveKey != "images/logo.png" || mock.moveTo != "images/brand/logo.png" {
		t.Errorf("store got app=%q key=%q to=%q", mock.levelApp, mock.moveKey, mock.moveTo)
	}
	if !strings.Contains(w.Body.String(), `"key":"images/brand/logo.png"`) {
		t.Errorf("move response = %s, want the new key", w.Body.String())
	}
}

// A taken destination is a 409; a body without a key fails validation
// before reaching the store.
func TestMoveAsset_Conflict(t *testing.T) {
	mock := &mockAssets{err: stores.ErrAssetExists}
	w := assetDriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/assets/object/move?key=a.png",
		map[string]any{"key": "b.png"})
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	mock = &mockAssets{}
	w = assetDriver(t, mock).Request(t, http.MethodPost, "/apps/app-1/assets/object/move?key=a.png",
		map[string]any{})
	if w.Code != http.StatusUnprocessableEntity || mock.moveKey != "" {
		t.Errorf("empty body: status = %d (want 422), store called = %v", w.Code, mock.moveKey != "")
	}
}

func TestDeleteAsset_OK(t *testing.T) {
	mock := &mockAssets{}
	w := assetDriver(t, mock).Request(t, http.MethodDelete, "/apps/app-1/assets/object?key=old.png", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.delApp != "app-1" || mock.delKey != "old.png" {
		t.Errorf("store got delete app=%q key=%q, want app-1/old.png", mock.delApp, mock.delKey)
	}
}

func TestDeleteAsset_NotFound(t *testing.T) {
	w := assetDriver(t, &mockAssets{err: stores.ErrNotFound}).Request(t, http.MethodDelete, "/apps/app-1/assets/object?key=missing", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// The published surface serves the live bytes — same bucket read, no authoring
// scope required.
func TestGetPublishedAsset_OK(t *testing.T) {
	mock := &mockAssets{asset: &models.Asset{Key: "images/logo.png", ContentType: "image/png", Data: []byte("PNG")}}
	w := assetDriver(t, mock).RequestRaw(t, testkit.DefaultTenant,
		http.MethodGet, "/published/apps/app-1/assets/object?key=images/logo.png", "", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.getApp != "app-1" || mock.getKey != "images/logo.png" {
		t.Errorf("store got app=%q key=%q", mock.getApp, mock.getKey)
	}
	if !bytes.Equal(w.Body.Bytes(), []byte("PNG")) || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("body=%q ct=%q, want the raw bytes as image/png", w.Body.Bytes(), w.Header().Get("Content-Type"))
	}
}

func TestGetPublishedAsset_NotFound(t *testing.T) {
	w := assetDriver(t, &mockAssets{err: stores.ErrNotFound}).RequestRaw(t, testkit.DefaultTenant,
		http.MethodGet, "/published/apps/app-1/assets/object?key=missing", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// A store failure on the folder and stats reads goes through the error
// transformer: an unrecognized error renders as a 500.
func TestListAssetFolder_Error(t *testing.T) {
	w := assetDriver(t, &mockAssets{err: errors.New("bucket down")}).RequestRaw(t, testkit.DefaultTenant,
		http.MethodGet, "/apps/app-1/assets/folder?path=images", "", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAssetStats_Error(t *testing.T) {
	w := assetDriver(t, &mockAssets{err: errors.New("db down")}).RequestRaw(t, testkit.DefaultTenant,
		http.MethodGet, "/apps/app-1/assets/stats", "", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}
