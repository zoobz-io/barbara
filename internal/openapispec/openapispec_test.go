package openapispec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoobz-io/openapi"
	"github.com/zoobz-io/rocco"

	adminhandlers "github.com/zoobz-io/barbara/admin/handlers"
	apihandlers "github.com/zoobz-io/barbara/api/handlers"
)

// dumpAndParse runs Dump into a temp file and unmarshals the result.
func dumpAndParse(t *testing.T, configure func(*rocco.Engine), endpoints []rocco.Endpoint) *openapi.OpenAPI {
	t.Helper()
	out := filepath.Join(t.TempDir(), "sub", "openapi.json")
	if err := Dump(configure, endpoints, out); err != nil {
		t.Fatalf("Dump: %v", err)
	}
	data, err := os.ReadFile(out) //nolint:gosec // test-owned temp path
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec openapi.OpenAPI
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}
	return &spec
}

func TestDumpAPISpec(t *testing.T) {
	spec := dumpAndParse(t, apihandlers.ConfigureOpenAPI, apihandlers.All())

	if spec.Info.Title != "Barbara API" {
		t.Errorf("title = %q, want Barbara API", spec.Info.Title)
	}
	if len(spec.Paths) == 0 {
		t.Error("spec has no paths")
	}
	// The patch: rocco $refs ValidationFieldError but never defines it.
	if _, ok := spec.Components.Schemas["ValidationFieldError"]; !ok {
		t.Error("components missing backfilled ValidationFieldError schema")
	}
}

func TestDumpAdminSpec(t *testing.T) {
	spec := dumpAndParse(t, adminhandlers.ConfigureOpenAPI, adminhandlers.All())

	if spec.Info.Title != "Barbara Admin API" {
		t.Errorf("title = %q, want Barbara Admin API", spec.Info.Title)
	}
	if len(spec.Paths) == 0 {
		t.Error("spec has no paths")
	}
}
