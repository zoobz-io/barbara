//go:build testing

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/zoobz-io/grub"
	grubopensearch "github.com/zoobz-io/grub/opensearch"
	osrenderer "github.com/zoobz-io/lucene/opensearch"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

// seedApp creates an app for the ctx tenant — the serving unit every published
// read scopes by, and the placement target every seeded document needs. The
// name is random because the fixed test tenants live in a persistent database:
// a fixed name would collide on the second run.
func seedApp(t *testing.T, st *stores.Stores, ctx context.Context) *models.App {
	t.Helper()
	app, err := st.Apps.Create(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("seeding app: %v", err)
	}
	return app
}

// seedDoc inserts a keyed document placed at the given app's root (no
// collection) for tests that just need a document to exist — publishing, tags,
// reindex, and the like, which key off the document's key and published
// pointer, not its tree position. The collection-aware create is exercised by
// the documents tree tests; this seeds via the store's generic insert, which
// the authoring create no longer exposes for raw keys. The tenant comes from
// ctx.
func seedDoc(st *stores.Stores, ctx context.Context, appID, key string) (*models.Document, error) {
	tenant, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	name := key
	if i := strings.LastIndex(key, "/"); i >= 0 {
		name = key[i+1:]
	}
	return st.Documents.Insert().Exec(ctx, &models.Document{
		TenantID: tenant, AppID: appID, Name: name,
		Key: key, Tags: pq.StringArray{}, CreatedAt: now, UpdatedAt: now,
	})
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// integrationSkip skips a test when a backing service is unreachable — so the
// suite is a no-op on a machine without the dev stack — EXCEPT in CI, where the
// stack is provisioned as service containers and an unreachable service is a
// real failure, not a reason to silently pass. GitHub Actions sets CI=true.
func integrationSkip(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("CI requires the integration stack but "+format, args...)
	}
	t.Skipf(format, args...)
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
		integrationSkip(t, "Postgres not reachable (%v)", err)
	}
	return db
}

// testUser is the acting user integration tests run as — a valid UUID, since
// created_by columns are UUID.
const testUser = "99999999-9999-9999-9999-999999999999"

// tenantCtx returns a context scoped to the given tenant, the way a handler
// bridges req.Identity before calling a store.
func tenantCtx(tenantID string) context.Context {
	p := auth.NewPrincipal(testUser, tenantID, "", nil, nil)
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
		integrationSkip(t, "OpenSearch not reachable at %s (%v)", addr, err)
	}
	_ = resp.Body.Close()

	client, err := opensearch.NewClient(opensearch.Config{Addresses: []string{addr}})
	if err != nil {
		t.Fatalf("opensearch client: %v", err)
	}
	return grubopensearch.New(client, grubopensearch.Config{Version: osrenderer.V2})
}
