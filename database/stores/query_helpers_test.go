//go:build testing

// Query-generation unit tests. These assert, against grub's mock SQL driver,
// that each store's soy builders emit the SQL we expect — tenant scoping, tag
// containment, keyset paging, FOR UPDATE / SKIP LOCKED, etc. — without a live
// Postgres. The mock (github.com/zoobz-io/grub/mockdb, exposed in grub v1.0.19)
// captures the generated SQL and args at the driver boundary; a method's return
// value is irrelevant here (reads see no rows and error out), so the tests drive
// the method and inspect the capture.

package stores

import (
	"context"
	"fmt"
	"strings"
	"testing"

	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/grub/mockdb"

	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/testing/testkit"
)

const (
	testTenant = "t-1"
	testUser   = "11111111-1111-1111-1111-111111111111"
)

// newQueryTest builds the Stores aggregate over grub's mock SQL driver and
// returns it with the query capture. sum.Reset()/sum.New() clears the scio
// catalog first, so each test re-registers the store tables without a
// duplicate-registration panic.
func newQueryTest(t *testing.T) (*Stores, *mockdb.Capture) {
	t.Helper()
	sum.Reset()
	sum.New()
	db, capture := mockdb.New()
	return New(db, astqlpg.New(), testkit.NewSearchProvider()), capture
}

// tenantCtx returns a context carrying a principal for the test tenant and user,
// the way a handler bridges req.Identity before calling a tenant-scoped store.
func tenantCtx() context.Context {
	return auth.WithPrincipal(context.Background(),
		auth.NewPrincipal(testUser, testTenant, "", nil, nil))
}

// lastQuery returns the most recently captured query, failing if none ran.
func lastQuery(t *testing.T, capture *mockdb.Capture) mockdb.CapturedQuery {
	t.Helper()
	q, ok := capture.Last()
	if !ok {
		t.Fatal("no query was captured")
	}
	return q
}

// wantSQL asserts the captured SQL contains every fragment (in astql's emitted,
// double-quoted form). Reports the full SQL on any miss.
func wantSQL(t *testing.T, q mockdb.CapturedQuery, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if !strings.Contains(q.Query, f) {
			t.Errorf("query missing %q\n  full: %s", f, q.Query)
		}
	}
}

// notSQL asserts the captured SQL does NOT contain the fragment — e.g. a
// deliberately tenant-agnostic reindex read must not carry tenant scoping.
func notSQL(t *testing.T, q mockdb.CapturedQuery, fragment string) {
	t.Helper()
	if strings.Contains(q.Query, fragment) {
		t.Errorf("query unexpectedly contains %q\n  full: %s", fragment, q.Query)
	}
}

// wantArg asserts the captured args bind a value equal to want.
func wantArg(t *testing.T, q mockdb.CapturedQuery, want any) {
	t.Helper()
	target := fmt.Sprintf("%v", want)
	for _, a := range q.Args {
		if fmt.Sprintf("%v", a) == target {
			return
		}
	}
	t.Errorf("args %v missing %v\n  full: %s", q.Args, want, q.Query)
}
