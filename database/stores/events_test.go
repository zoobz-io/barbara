//go:build testing

package stores

import (
	"context"
	"errors"
	"testing"

	"github.com/zoobz-io/grub/mockdb"

	"github.com/zoobz-io/barbara/events"
)

// A representative success path: Documents.Create emits Document.Created once the
// insert has committed, carrying the created id, tenant, and key.
func TestEvents_DocumentCreated_OnSuccess(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())    // requireParentScope: apps.Get
	cfg.PushRowData(countRow(0)) // siblingCollectionExists
	cfg.PushRowData(docRow(nil)) // the INSERT ... RETURNING scans a created row

	var got events.DocumentCreatedEvent
	fired := false
	l := events.Document.Created.Listen(func(_ context.Context, e events.DocumentCreatedEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if _, err := st.Documents.Create(tenantCtx(), testApp, nil, "a.md"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !fired {
		t.Fatal("Document.Created was not emitted on success")
	}
	if got.DocumentID != "d-1" || got.TenantID != testTenant || got.Key != "a.md" {
		t.Errorf("event payload = %+v, want the created document's id/tenant/key", got)
	}
}

// The mirror: when the insert fails, no event fires — an event never stands in
// for work that did not commit.
func TestEvents_DocumentCreated_SilentOnFailure(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())                    // scope ok
	cfg.PushRowData(countRow(0))                 // no sibling collection
	cfg.PushQueryErr(errors.New("insert boom"))  // the INSERT errors

	fired := false
	l := events.Document.Created.Listen(func(_ context.Context, _ events.DocumentCreatedEvent) {
		fired = true
	})
	defer l.Close()

	if _, err := st.Documents.Create(tenantCtx(), testApp, nil, "a.md"); err == nil {
		t.Fatal("expected create to fail")
	}
	if fired {
		t.Error("Document.Created was emitted despite a failed insert")
	}
}

// A no-op tag change (adding a tag the document already carries) commits nothing
// and must emit nothing — the emission is gated on an actual change.
func TestEvents_TagAdded_SilentOnNoOp(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	// The lock select returns a document that already carries "guide" (and is a
	// draft), so AddTag("guide") is a no-op: it writes nothing and, having not
	// changed, emits nothing.
	cfg.PushRowData(&mockdb.RowData{
		Columns: []string{"id", "tenant_id", "key", "tags"},
		Rows:    [][]any{{"d-1", testTenant, "a.md", "{guide}"}},
	})

	fired := false
	l := events.Document.TagAdded.Listen(func(_ context.Context, _ events.DocumentTagAddedEvent) {
		fired = true
	})
	defer l.Close()

	if _, err := st.AddTag(tenantCtx(), "d-1", "guide"); err != nil {
		t.Fatalf("add tag: %v", err)
	}
	if fired {
		t.Error("Document.TagAdded emitted for a no-op re-add")
	}
}
