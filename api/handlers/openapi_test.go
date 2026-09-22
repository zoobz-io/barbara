package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/zoobz-io/barbara/internal/openapispec"
)

// TestMdastSchemaMatchesFixtures validates every golden mdast fixture against
// the MdastRoot schema in the generated OpenAPI spec. It is the guard against
// drift between the Go node structs in internal/mdast and the schema the SDK is
// generated from: change a struct without the schema following (or vice versa)
// and a fixture stops validating.
func TestMdastSchemaMatchesFixtures(t *testing.T) {
	// Generate the real spec the SDK is built from, including the openapispec
	// post-processing (free-form meta, mdast nullability).
	specPath := filepath.Join(t.TempDir(), "openapi.json")
	if err := openapispec.Dump(ConfigureOpenAPI, All(), specPath); err != nil {
		t.Fatalf("dump spec: %v", err)
	}
	spec := readJSON(t, specPath)

	c := jsonschema.NewCompiler()
	if err := c.AddResource("openapi.json", spec); err != nil {
		t.Fatalf("add spec resource: %v", err)
	}
	root, err := c.Compile("openapi.json#/components/schemas/Root")
	if err != nil {
		t.Fatalf("compile Root schema: %v", err)
	}

	fixtures, err := filepath.Glob(filepath.Join("..", "..", "internal", "mdast", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no mdast fixtures found")
	}
	for _, f := range fixtures {
		name := filepath.Base(f)
		// Skip the sidecar frontmatter metadata files; they are not mdast trees.
		if len(name) > len(".meta.json") && name[len(name)-len(".meta.json"):] == ".meta.json" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			inst := readJSON(t, f)
			if err := root.Validate(inst); err != nil {
				t.Errorf("fixture does not validate against the mdast Root schema:\n%v", err)
			}
		})
	}
}

func readJSON(t *testing.T, path string) any {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // test-owned generated/fixture path
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	v, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return v
}
