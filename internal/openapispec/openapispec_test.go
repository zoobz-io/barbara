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

// Dump fails when the output directory cannot be created (a regular file is
// in the way) or the spec cannot be written (the path is a directory).
func TestDump_OutputErrors(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Dump(apihandlers.ConfigureOpenAPI, apihandlers.All(), filepath.Join(blocker, "openapi.json")); err == nil {
		t.Error("Dump beneath a regular file succeeded")
	}
	if err := Dump(apihandlers.ConfigureOpenAPI, apihandlers.All(), dir); err == nil {
		t.Error("Dump onto a directory succeeded")
	}
}

// patch backfills ValidationFieldError only when there is a schema map to
// put it in and nothing already defines it.
func TestPatch(t *testing.T) {
	patch(&openapi.OpenAPI{}) // no components: nothing to do

	spec := &openapi.OpenAPI{Components: &openapi.Components{}}
	patch(spec)
	if spec.Components.Schemas != nil {
		t.Errorf("patch invented a schema map: %v", spec.Components.Schemas)
	}

	existing := &openapi.Schema{Description: "already defined"}
	spec = &openapi.OpenAPI{Components: &openapi.Components{
		Schemas: map[string]*openapi.Schema{"ValidationFieldError": existing},
	}}
	patch(spec)
	if spec.Components.Schemas["ValidationFieldError"] != existing {
		t.Error("patch replaced an existing ValidationFieldError schema")
	}

	spec = &openapi.OpenAPI{Components: &openapi.Components{Schemas: map[string]*openapi.Schema{}}}
	patch(spec)
	got := spec.Components.Schemas["ValidationFieldError"]
	if got == nil || len(got.Required) != 2 || got.Properties["field"] == nil || got.Properties["message"] == nil {
		t.Errorf("backfilled schema = %+v", got)
	}
}
