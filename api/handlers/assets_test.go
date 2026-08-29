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
	asset   *models.Asset
	list    []*models.Asset
	err     error
	putKey  string
	putCT   string
	putData []byte
	getKey  string
	delKey  string
}

func (m *mockAssets) Put(_ context.Context, key, contentType string, data []byte) (*models.Asset, error) {
	m.putKey, m.putCT, m.putData = key, contentType, data
	if m.err != nil {
		return nil, m.err
	}
	if m.asset != nil {
		return m.asset, nil
	}
	return &models.Asset{Key: key, ContentType: contentType, Size: int64(len(data))}, nil
}

func (m *mockAssets) Get(_ context.Context, key string) (*models.Asset, error) {
	m.getKey = key
	return m.asset, m.err
}

func (m *mockAssets) List(context.Context) ([]*models.Asset, error) { return m.list, m.err }

func (m *mockAssets) Delete(_ context.Context, key string) error {
	m.delKey = key
	return m.err
}

func assetDriver(t *testing.T, mock contracts.Assets) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Assets](k, mock)
	}, All()...)
}

// Upload passes the raw body and its content type through to the store and
// echoes back the stored metadata.
func TestUploadAsset_OK(t *testing.T) {
	mock := &mockAssets{}
	w := assetDriver(t, mock).RequestRaw(t, testkit.DefaultTenant,
		http.MethodPut, "/assets/object?key=images/logo.png", "image/png", []byte("PNGBYTES"))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.putKey != "images/logo.png" || mock.putCT != "image/png" || !bytes.Equal(mock.putData, []byte("PNGBYTES")) {
		t.Errorf("store got key=%q ct=%q data=%q", mock.putKey, mock.putCT, mock.putData)
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
		http.MethodGet, "/assets/object?key=doc.pdf", "", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.getKey != "doc.pdf" {
		t.Errorf("store got key %q, want doc.pdf", mock.getKey)
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
		http.MethodGet, "/assets/object?key=missing", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestListAssets_OK(t *testing.T) {
	mock := &mockAssets{list: []*models.Asset{
		{Key: "a.png", ContentType: "image/png", Size: 10},
		{Key: "b.pdf", ContentType: "application/pdf", Size: 20},
	}}
	w := assetDriver(t, mock).Request(t, http.MethodGet, "/assets", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Assets []struct {
			Key  string `json:"key"`
			Size int64  `json:"size"`
		} `json:"assets"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 || len(resp.Assets) != 2 || resp.Assets[0].Key != "a.png" {
		t.Errorf("list response = %s", w.Body.String())
	}
}

func TestDeleteAsset_OK(t *testing.T) {
	mock := &mockAssets{}
	w := assetDriver(t, mock).Request(t, http.MethodDelete, "/assets/object?key=old.png", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if mock.delKey != "old.png" {
		t.Errorf("store got delete key %q, want old.png", mock.delKey)
	}
}

func TestDeleteAsset_NotFound(t *testing.T) {
	w := assetDriver(t, &mockAssets{err: stores.ErrNotFound}).Request(t, http.MethodDelete, "/assets/object?key=missing", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}
