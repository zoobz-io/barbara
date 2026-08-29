//go:build testing

package integration

import (
	"context"
	"errors"
	"sync"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

// versionsFixture creates a parent document and returns the stores plus its ID.
func versionsFixture(t *testing.T) (*stores.Versions, string) {
	t.Helper()
	db := pgDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM versions")
		_, _ = db.Exec("DELETE FROM documents")
		_ = db.Close()
	})
	st := stores.New(db, astqlpg.New(), testkit.NewSearchProvider(), testkit.NewBucketProvider())
	doc, err := seedDoc(st, tenantCtx(testTenant), seedApp(t, st, tenantCtx(testTenant)).ID, "versioned.md")
	if err != nil {
		t.Fatalf("create parent document: %v", err)
	}
	return st.Versions, doc.ID
}

func TestVersions_SaveListGet(t *testing.T) {
	versions, docID := versionsFixture(t)
	ctx := tenantCtx(testTenant)

	v1, err := versions.Save(ctx, docID, "first", 0)
	if err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if v1.VersionNumber != 1 {
		t.Errorf("first version number = %d, want 1", v1.VersionNumber)
	}
	if v1.CreatedBy != testUser {
		t.Errorf("created_by = %q, want %q", v1.CreatedBy, testUser)
	}

	v2, err := versions.Save(ctx, docID, "second", 1)
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if v2.VersionNumber != 2 {
		t.Errorf("second version number = %d, want 2", v2.VersionNumber)
	}

	// Get is tenant-scoped.
	got, err := versions.Get(ctx, v1.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Content != "first" {
		t.Errorf("content = %q, want first", got.Content)
	}
	if _, err := versions.Get(tenantCtx(otherTenant), v1.ID); !errors.Is(err, stores.ErrNotFound) {
		t.Errorf("cross-tenant get = %v, want ErrNotFound", err)
	}

	// List is newest-first.
	list, err := versions.List(ctx, docID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].VersionNumber != 2 || list[1].VersionNumber != 1 {
		t.Errorf("list order wrong: %+v", list)
	}
}

func TestVersions_Save_RequiresUser(t *testing.T) {
	versions, docID := versionsFixture(t)

	// A tenant but no acting user — every version records who created it.
	ctx := auth.WithPrincipal(context.Background(), auth.NewPrincipal("", testTenant, "", nil, nil))
	if _, err := versions.Save(ctx, docID, "x", 0); !errors.Is(err, auth.ErrNoUser) {
		t.Errorf("save without user = %v, want ErrNoUser", err)
	}

	// No tenant at all is also refused.
	if _, err := versions.Save(context.Background(), docID, "x", 0); !errors.Is(err, auth.ErrNoTenant) {
		t.Errorf("save without tenant = %v, want ErrNoTenant", err)
	}
}

// Optimistic concurrency: N concurrent saves from the same base race for
// the head. Exactly one wins; the rest conflict, reporting the current head — so
// two editors never silently clobber each other. Only the winner's version
// persists.
func TestVersions_ConcurrentSaves_OneWinnerRestConflict(t *testing.T) {
	versions, docID := versionsFixture(t)
	ctx := tenantCtx(testTenant)

	const n = 12
	var wg sync.WaitGroup
	var mu sync.Mutex
	var winners, conflicts, wrongHead, otherErr int

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := versions.Save(ctx, docID, "concurrent", 0) // all edit from the empty doc
			var vce *stores.VersionConflictError
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				winners++
			case errors.As(err, &vce):
				conflicts++
				if vce.CurrentHead != 1 {
					wrongHead++ // the winner made head 1; every conflict should report it
				}
			default:
				otherErr++
			}
		}()
	}
	wg.Wait()

	if otherErr != 0 {
		t.Fatalf("%d concurrent saves failed with unexpected errors", otherErr)
	}
	if winners != 1 {
		t.Errorf("winners = %d, want exactly 1", winners)
	}
	if conflicts != n-1 {
		t.Errorf("conflicts = %d, want %d", conflicts, n-1)
	}
	if wrongHead != 0 {
		t.Errorf("%d conflicts reported a head other than 1 (the winner)", wrongHead)
	}

	// Exactly one version persisted — the winner's, numbered 1.
	list, err := versions.List(ctx, docID, 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].VersionNumber != 1 {
		t.Errorf("persisted %d versions, want exactly one (v1): %+v", len(list), list)
	}
}
