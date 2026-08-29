//go:build testing

package stores

import (
	"errors"
	"testing"

	"github.com/zoobz-io/grub/mockdb"
)

// Publish validates the version belongs to the document before cutting anything.
func TestPublish_ValidatesVersionOwnership(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(docRow(testApp)) // Documents.Get: a placed document
	cfg.PushRowData(&mockdb.RowData{ // Versions.Get: belongs to a different document
		Columns: []string{"id", "document_id", "tenant_id", "version_number", "content"},
		Rows:    [][]any{{"v-1", "other-doc", testTenant, int64(1), "x"}},
	})

	if _, err := st.Publish(tenantCtx(), "d-1", "v-1"); !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("publish a foreign version = %v, want ErrVersionMismatch", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `INSERT INTO "releases"`)
	}
}

// The document is loaded first, tenant-scoped.
func TestPublish_LoadsDocumentFirst(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Publish(tenantCtx(), "d-1", "v-1")

	q := queryAt(t, capture, 0)
	wantSQL(t, q, `FROM "documents"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "d-1")
	wantArg(t, q, testTenant)
}
