package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/zoobz-io/barbara/internal/openapispec"
)

// TestMdastSchemaMatchesFixtures validates every golden fixture against the
// generated OpenAPI schema for its tree: the .json fixtures against MdastRoot
// and the .hast.json fixtures against HastRoot. It is the guard against drift
// between the Go node structs (internal/mdast, internal/hast) and the schemas
// the SDK is generated from: change a struct without the schema following (or
// vice versa) and a fixture stops validating.
func TestMdastSchemaMatchesFixtures(t *testing.T) {
	// Generate the real spec the SDK is built from, including the openapispec
	// post-processing (free-form meta and properties, mdast nullability).
	specPath := filepath.Join(t.TempDir(), "openapi.json")
	if err := openapispec.Dump(ConfigureOpenAPI, All(), specPath); err != nil {
		t.Fatalf("dump spec: %v", err)
	}
	spec := readJSON(t, specPath)

	c := jsonschema.NewCompiler()
	if err := c.AddResource("openapi.json", spec); err != nil {
		t.Fatalf("add spec resource: %v", err)
	}
	compile := func(name string) *jsonschema.Schema {
		s, err := c.Compile("openapi.json#/components/schemas/" + name)
		if err != nil {
			t.Fatalf("compile %s schema: %v", name, err)
		}
		return s
	}

	testdata := filepath.Join("..", "..", "internal", "mdast", "testdata")
	// The mdast tree (<name>.json) validates against MdastRoot; the hast tree
	// (<name>.hast.json) against HastRoot. Match .hast.json first so it is not
	// caught by the .json glob.
	validate(t, compile("HastRoot"), filepath.Join(testdata, "*.hast.json"), nil)
	validate(t, compile("MdastRoot"), filepath.Join(testdata, "*.json"), func(name string) bool {
		// Skip hast trees (also *.json) and the sidecar frontmatter files.
		return strings.HasSuffix(name, ".hast.json") || strings.HasSuffix(name, ".meta.json")
	})
}

// validate compiles the fixtures matching pattern and checks each against schema,
// skipping any whose base name skip reports true.
func validate(t *testing.T, schema *jsonschema.Schema, pattern string, skip func(name string) bool) {
	t.Helper()
	fixtures, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatalf("no fixtures matched %s", pattern)
	}
	for _, f := range fixtures {
		name := filepath.Base(f)
		if skip != nil && skip(name) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			if err := schema.Validate(readJSON(t, f)); err != nil {
				t.Errorf("fixture does not validate against its generated schema:\n%v", err)
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
