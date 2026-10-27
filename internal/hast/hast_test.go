package hast_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoobz-io/barbara/internal/hast"
	"github.com/zoobz-io/barbara/internal/mdast"
	"github.com/zoobz-io/barbara/internal/treejson"
)

// testdata is internal/mdast/testdata: the .md fixtures and their golden trees
// are shared between the two packages, so both read from there.
var testdata = filepath.Join("..", "mdast", "testdata")

// TestFixtures parses every .md fixture to mdast, converts it to hast, and
// compares the JSON, byte for byte, with the golden .hast.json produced by
// mdast-util-to-hast (web/scripts/mdast-fixtures.mjs). This is the guarantee
// that FromMdast reproduces the library the client renders with.
func TestFixtures(t *testing.T) {
	mds, err := filepath.Glob(filepath.Join(testdata, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, md := range mds {
		if filepath.Base(md) == "README.md" {
			continue
		}
		name := base(md)
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(md) //nolint:gosec // test-owned fixture path
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join(testdata, name+".hast.json")) //nolint:gosec // test-owned fixture path
			if err != nil {
				t.Fatalf("missing golden file: %v", err)
			}
			root, _, err := mdast.Parse(src)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got, err := treejson.Canonical(hast.FromMdast(root))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("hast JSON differs from golden:\n%s", treejson.FirstDiff(want, got))
			}
		})
	}
}

func base(path string) string {
	b := filepath.Base(path)
	return b[:len(b)-len(filepath.Ext(b))]
}
