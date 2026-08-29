//go:build testing

package handlers

import (
	"net/http"
	"testing"

	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/api/contracts"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// A write endpoint refuses an identity that holds only the read scope.
func TestScopeGate_WriteRequired(t *testing.T) {
	reader := auth.NewStub("u", "t", "", nil, []string{auth.ScopeDocumentsRead})
	d := testkit.HandlersAs(t, reader, func(k sum.Key) {
		sum.Register[contracts.Documents](k, &mockDocuments{})
	}, All()...)

	w := d.Request(t, http.MethodPost, "/documents", map[string]string{"key": "a.md"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("create with read-only scope = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// A read endpoint refuses an identity that holds only the write scope.
func TestScopeGate_ReadRequired(t *testing.T) {
	writer := auth.NewStub("u", "t", "", nil, []string{auth.ScopeDocumentsWrite})
	d := testkit.HandlersAs(t, writer, func(k sum.Key) {
		sum.Register[contracts.Documents](k, &mockDocuments{})
	}, All()...)

	w := d.Request(t, http.MethodGet, "/documents/x", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("get with write-only scope = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// A publish endpoint refuses an identity that holds read+write but not publish.
func TestScopeGate_PublishRequired(t *testing.T) {
	noPublish := auth.NewStub("u", "t", "", nil, []string{auth.ScopeDocumentsRead, auth.ScopeDocumentsWrite})
	d := testkit.HandlersAs(t, noPublish, func(k sum.Key) {
		sum.Register[contracts.Publishing](k, &mockPublishing{})
	}, All()...)

	w := d.Request(t, http.MethodPost, "/documents/x/publish", map[string]string{"version_id": "v"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("publish without publish scope = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// Published reads are the site's render path — plain tenant auth, no scope. An
// identity with no scopes at all still reaches them (not a 403).
func TestScopeGate_PublishedReadsUngated(t *testing.T) {
	noScopes := auth.NewStub("u", "t", "", nil, nil)
	d := testkit.HandlersAs(t, noScopes, func(k sum.Key) {
		sum.Register[contracts.Reads](k, &mockReads{})
	}, All()...)

	w := d.Request(t, http.MethodGet, "/published/documents", nil)
	if w.Code == http.StatusForbidden {
		t.Fatalf("published reads were scope-gated (403); they should be ungated. body=%s", w.Body.String())
	}
}
