//go:build testing

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/zoobz-io/grub"
	grubopensearch "github.com/zoobz-io/grub/opensearch"
	osrenderer "github.com/zoobz-io/lucene/opensearch"
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

// osProvider builds the OpenSearch provider for integration tests, skipping
// when no cluster is reachable — so the suite is a no-op without the dev stack.
func osProvider(t *testing.T) grub.SearchProvider {
	t.Helper()
	addr := env("APP_OPENSEARCH_ADDR", "http://localhost:9200")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+"/_cluster/health", nil)
	if err != nil {
		t.Fatalf("building health request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("OpenSearch not reachable at %s (%v); skipping", addr, err)
	}
	_ = resp.Body.Close()

	client, err := opensearch.NewClient(opensearch.Config{Addresses: []string{addr}})
	if err != nil {
		t.Fatalf("opensearch client: %v", err)
	}
	return grubopensearch.New(client, grubopensearch.Config{Version: osrenderer.V2})
}
