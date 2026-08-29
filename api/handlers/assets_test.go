//go:build testing

package handlers

import (
	"bytes"
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
			Size int64  `json:"size"`
		} `json:"assets"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 || len(resp.Assets) != 2 || resp.Assets[0].Key != "images/a.png" {
		t.Errorf("list response = %s", w.Body.String())
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
