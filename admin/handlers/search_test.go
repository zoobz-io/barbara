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
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// mockSearch is a contracts.Search whose behavior each test sets.
type mockSearch struct {
	results []models.DocumentIndex
	total   int64
	err     error
	gotQ    string
}

func (m *mockSearch) SearchAll(_ context.Context, query string, _, _ int) ([]models.DocumentIndex, int64, error) {
	m.gotQ = query
	return m.results, m.total, m.err
}

func searchDriver(t *testing.T, mock contracts.Search) *testkit.Driver {
	return testkit.Handlers(t, func(k sum.Key) {
		sum.Register[contracts.Search](k, mock)
	}, All()...)
}

// A cross-tenant search returns hits with their owning tenant. The default stub
// holds the admin role, so the gate passes.
func TestSearchAll_OK(t *testing.T) {
	mock := &mockSearch{
		results: []models.DocumentIndex{{DocumentID: "d1", TenantID: "t1", Key: "a.md", Content: "install"}},
		total:   1,
	}
	w := searchDriver(t, mock).Request(t, http.MethodGet, "/search?q=install", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if mock.gotQ != "install" {
		t.Errorf("store got query %q, want install", mock.gotQ)
	}
	var resp struct {
		Total   int64 `json:"total"`
		Results []struct {
			DocumentID string `json:"document_id"`
			TenantID   string `json:"tenant_id"`
		} `json:"results"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 || len(resp.Results) != 1 || resp.Results[0].TenantID != "t1" {
		t.Errorf("search response = %s, want one cross-tenant hit exposing tenant_id", w.Body.String())
	}
}

func TestSearchAll_MissingQuery(t *testing.T) {
	w := searchDriver(t, &mockSearch{}).Request(t, http.MethodGet, "/search", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

// An identity without the admin role is refused with 403 — the admin gate holds.
func TestSearchAll_ForbiddenWithoutAdminRole(t *testing.T) {
	nonAdmin := auth.NewStub("u-1", "t-1", "", nil, nil) // no roles, no scopes
	d := testkit.HandlersAs(t, nonAdmin, func(k sum.Key) {
		sum.Register[contracts.Search](k, &mockSearch{})
	}, All()...)

	w := d.Request(t, http.MethodGet, "/search?q=install", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (no admin role); body=%s", w.Code, w.Body.String())
	}
}
