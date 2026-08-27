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
	docs := stores.NewDocuments(db, astqlpg.New())
	versions := stores.NewVersions(db, astqlpg.New(), docs)
	doc, err := docs.Create(tenantCtx(testTenant), "versioned.md")
	if err != nil {
		t.Fatalf("create parent document: %v", err)
	}
	return versions, doc.ID
}

func TestVersions_SaveListGet(t *testing.T) {
	versions, docID := versionsFixture(t)
	ctx := tenantCtx(testTenant)

	v1, err := versions.Save(ctx, docID, "first")
	if err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if v1.VersionNumber != 1 {
		t.Errorf("first version number = %d, want 1", v1.VersionNumber)
	}
	if v1.CreatedBy != testUser {
		t.Errorf("created_by = %q, want %q", v1.CreatedBy, testUser)
	}

	v2, err := versions.Save(ctx, docID, "second")
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
	if _, err := versions.Save(ctx, docID, "x"); !errors.Is(err, stores.ErrNoUser) {
		t.Errorf("save without user = %v, want ErrNoUser", err)
	}

	// No tenant at all is also refused.
	if _, err := versions.Save(context.Background(), docID, "x"); !errors.Is(err, stores.ErrNoTenant) {
		t.Errorf("save without tenant = %v, want ErrNoTenant", err)
	}
}

// The race the plan calls out: concurrent saves for the same document must all
// persist, each with a distinct monotonic version_number.
func TestVersions_ConcurrentSaves_AllPersistDistinct(t *testing.T) {
	versions, docID := versionsFixture(t)
	ctx := tenantCtx(testTenant)

	const n = 12
	var wg sync.WaitGroup
	var mu sync.Mutex
	nums := map[int]bool{}
	errs := 0

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := versions.Save(ctx, docID, "concurrent")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs++
				return
			}
			nums[v.VersionNumber] = true
		}()
	}
	wg.Wait()

	if errs != 0 {
		t.Fatalf("%d concurrent saves failed", errs)
	}
	if len(nums) != n {
		t.Fatalf("got %d distinct version numbers, want %d (collision or loss)", len(nums), n)
	}
	// The numbers must be exactly 1..n with no gaps.
	for i := 1; i <= n; i++ {
		if !nums[i] {
			t.Errorf("missing version_number %d", i)
		}
	}
}
