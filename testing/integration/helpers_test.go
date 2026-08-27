//go:build testing

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/internal/auth"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// pgDB connects to Postgres for integration tests, resetting the sum catalog so
// stores can re-register, and skipping when the database or schema is absent —
// so the suite is a no-op without the dev stack.
func pgDB(t *testing.T) *sqlx.DB {
	t.Helper()
	sum.Reset()
	sum.New()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("APP_DB_HOST", "127.0.0.1"), env("APP_DB_PORT", "5432"),
		env("APP_DB_USER", "barbara"), env("APP_DB_PASSWORD", "barbara"),
		env("APP_DB_NAME", "barbara"))

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres not reachable (%v); skipping integration test", err)
	}
	return db
}

// tenantCtx returns a context scoped to the given tenant, the way a handler
// bridges req.Identity before calling a store.
func tenantCtx(tenantID string) context.Context {
	p := auth.NewPrincipal("user-1", tenantID, "", nil, nil)
	return auth.WithPrincipal(context.Background(), p)
}
