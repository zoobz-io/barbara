//go:build testing

package stores

import (
	"context"
	"errors"
	"testing"

	"github.com/lib/pq"

	"github.com/zoobz-io/grub/mockdb"

	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// Create inserts every non-PK column for the request's tenant and returns the
// generated row; the id is DB-generated, so it is omitted from the column list.
func TestApps_Create_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Apps.Create(tenantCtx(), "docs-site")

	q := lastQuery(t, capture)
	wantSQL(t, q, `INSERT INTO "apps"`, `"tenant_id"`, `"name"`, `RETURNING`)
	wantArg(t, q, testTenant)
	wantArg(t, q, "docs-site")
}

// Get is scoped to the request's tenant: id AND tenant_id.
func TestApps_Get_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Apps.Get(tenantCtx(), "app-1")

	q := lastQuery(t, capture)
	wantSQL(t, q, `FROM "apps"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, q, "app-1")
	wantArg(t, q, testTenant)
}

// List is tenant-scoped, oldest first, and paginated with LIMIT/OFFSET.
func TestApps_List_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Apps.List(tenantCtx(), 10, 5)

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "apps"`,
		`"tenant_id" = ?`,
		`ORDER BY "created_at" ASC`,
		`LIMIT 10`,
		`OFFSET 5`,
	)
}

// Rename updates name + updated_at, scoped to the tenant, and returns the row.
func TestApps_Rename_Query(t *testing.T) {
	st, capture := newQueryTest(t)
	_, _ = st.Apps.Rename(tenantCtx(), "app-1", "marketing-site")

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`UPDATE "apps" SET`,
		`"name" = ?`,
		`"updated_at" = ?`,
		`"id" = ?`,
		`"tenant_id" = ?`,
		`RETURNING`,
	)
	wantArg(t, q, "marketing-site")
	wantArg(t, q, testTenant)
}

// Delete first counts the app's releases (the guard), then — with none — issues
// a tenant-scoped DELETE.
func TestApps_Delete_Query(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(0)) // no releases: the guard passes

	_ = st.Apps.Delete(tenantCtx(), "app-1")

	count := queryAt(t, capture, 0)
	wantSQL(t, count, `FROM "releases"`, `"app_id" = ?`, `"tenant_id" = ?`)
	wantArg(t, count, "app-1")

	del := queryAt(t, capture, 1)
	wantSQL(t, del, `DELETE FROM "apps"`, `"id" = ?`, `"tenant_id" = ?`)
	wantArg(t, del, "app-1")
	wantArg(t, del, testTenant)
}

// Delete refuses an app that has any release: the guard count is non-zero, so it
// returns ErrAppHasReleases and never issues the DELETE.
func TestApps_Delete_RefusesWithReleases(t *testing.T) {
	st, capture, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(1)) // one release exists

	err := st.Apps.Delete(tenantCtx(), "app-1")
	if !errors.Is(err, ErrAppHasReleases) {
		t.Fatalf("delete with a release = %v, want ErrAppHasReleases", err)
	}
	for _, q := range capture.Queries {
		notSQL(t, q, `DELETE FROM "apps"`)
	}
}

// Every store method requires a tenant in context; a tenantless call is
// rejected before any query runs.
func TestApps_RequiresTenant(t *testing.T) {
	st, capture := newQueryTest(t)
	ctx := context.Background()
	if _, err := st.Apps.Create(ctx, "x"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Create tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Apps.Get(ctx, "app-1"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Get tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Apps.List(ctx, 10, 0); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("List tenantless = %v, want ErrNoTenant", err)
	}
	if _, err := st.Apps.Rename(ctx, "app-1", "y"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Rename tenantless = %v, want ErrNoTenant", err)
	}
	if err := st.Apps.Delete(ctx, "app-1"); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("Delete tenantless = %v, want ErrNoTenant", err)
	}
	if len(capture.Queries) != 0 {
		t.Errorf("a tenantless call issued %d queries, want 0", len(capture.Queries))
	}
}

// A release cut racing the delete surfaces at the DB as a foreign-key violation;
// the store translates it back to ErrAppHasReleases — the count guard's backstop.
func TestApps_Delete_ForeignKeyRaceIsHasReleases(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(0))              // guard passes: no releases at count time
	cfg.PushExecErr(&pq.Error{Code: "23503"}) // ...but the DELETE hits the FK (a cut raced in)

	if err := st.Apps.Delete(tenantCtx(), "app-1"); !errors.Is(err, ErrAppHasReleases) {
		t.Fatalf("delete racing a cut = %v, want ErrAppHasReleases", err)
	}
}

// A non-FK error from the DELETE is wrapped and returned, never swallowed as the
// release guard.
func TestApps_Delete_OtherErrorIsWrapped(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(0))
	cfg.PushExecErr(errors.New("delete boom"))

	err := st.Apps.Delete(tenantCtx(), "app-1")
	if err == nil || errors.Is(err, ErrAppHasReleases) {
		t.Fatalf("delete with a non-FK error = %v, want a wrapped error (not ErrAppHasReleases)", err)
	}
}

// A duplicate name (the per-tenant unique index) surfaces as ErrAppNameTaken,
// not a raw pq error, so the transformer maps it to a 409.
func TestApps_Create_DuplicateNameIsTaken(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushQueryErr(&pq.Error{Code: "23505"}) // unique violation on the INSERT

	_, err := st.Apps.Create(tenantCtx(), "docs-site")
	if !errors.Is(err, ErrAppNameTaken) {
		t.Fatalf("create with a duplicate name = %v, want ErrAppNameTaken", err)
	}
}

// Create emits App.Created once the insert commits, carrying the id/tenant/name.
func TestApps_Create_EmitsCreated(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(appRow())

	var got events.AppCreatedEvent
	fired := false
	l := events.App.Created.Listen(func(_ context.Context, e events.AppCreatedEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if _, err := st.Apps.Create(tenantCtx(), "docs-site"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !fired {
		t.Fatal("App.Created was not emitted on success")
	}
	if got.AppID != "app-1" || got.TenantID != testTenant || got.Name != "docs-site" {
		t.Errorf("event payload = %+v, want the created app's id/tenant/name", got)
	}
}

// No event fires when the insert fails — an event never stands in for work that
// did not commit.
func TestApps_Create_SilentOnFailure(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushQueryErr(errors.New("insert boom"))

	fired := false
	l := events.App.Created.Listen(func(_ context.Context, _ events.AppCreatedEvent) {
		fired = true
	})
	defer l.Close()

	if _, err := st.Apps.Create(tenantCtx(), "docs-site"); err == nil {
		t.Fatal("expected create to fail")
	}
	if fired {
		t.Error("App.Created was emitted despite a failed insert")
	}
}

// Rename emits App.Renamed carrying the new name.
func TestApps_Rename_EmitsRenamed(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(&mockdb.RowData{
		Columns: []string{"id", "tenant_id", "name"},
		Rows:    [][]any{{"app-1", testTenant, "marketing-site"}},
	})

	var got events.AppRenamedEvent
	fired := false
	l := events.App.Renamed.Listen(func(_ context.Context, e events.AppRenamedEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if _, err := st.Apps.Rename(tenantCtx(), "app-1", "marketing-site"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !fired {
		t.Fatal("App.Renamed was not emitted on success")
	}
	if got.AppID != "app-1" || got.Name != "marketing-site" {
		t.Errorf("event payload = %+v, want the renamed app's id/new name", got)
	}
}

// Delete emits App.Deleted once the row is gone (guard passed, one row removed).
func TestApps_Delete_EmitsDeleted(t *testing.T) {
	st, _, cfg := newQueryTestCfg(t)
	cfg.PushRowData(countRow(0)) // no releases: the guard passes; DELETE affects 1 row by default

	var got events.AppDeletedEvent
	fired := false
	l := events.App.Deleted.Listen(func(_ context.Context, e events.AppDeletedEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if err := st.Apps.Delete(tenantCtx(), "app-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !fired {
		t.Fatal("App.Deleted was not emitted on success")
	}
	if got.AppID != "app-1" || got.TenantID != testTenant {
		t.Errorf("event payload = %+v, want the deleted app's id/tenant", got)
	}
}
