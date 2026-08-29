//go:build testing

package stores

import (
	"context"
	"testing"
)

// splitKey derives tree placement from a path-like key: last segment is the
// name, earlier segments the collection path, empty segments dropped, and a
// degenerate all-slash key lands at the root under its verbatim key.
func TestSplitKey(t *testing.T) {
	cases := []struct {
		key      string
		wantPath []string
		wantName string
	}{
		{"readme.md", nil, "readme.md"},
		{"guides/install.md", []string{"guides"}, "install.md"},
		{"a/b/c.md", []string{"a", "b"}, "c.md"},
		{"a//b", []string{"a"}, "b"},
		{"/leading.md", nil, "leading.md"},
		{"trailing/", nil, "trailing"},
		{"///", nil, "///"},
	}
	for _, tc := range cases {
		path, name := splitKey(tc.key)
		if name != tc.wantName {
			t.Errorf("splitKey(%q) name = %q, want %q", tc.key, name, tc.wantName)
		}
		if len(path) != len(tc.wantPath) {
			t.Errorf("splitKey(%q) path = %v, want %v", tc.key, path, tc.wantPath)
			continue
		}
		for i := range path {
			if path[i] != tc.wantPath[i] {
				t.Errorf("splitKey(%q) path = %v, want %v", tc.key, path, tc.wantPath)
				break
			}
		}
	}
}

// The tenant scan walks unplaced documents (app_id IS NULL) by keyset, not
// OFFSET, so a large table stays linear. With no rows the backfill is a no-op.
func TestBackfill002_ScanQuery(t *testing.T) {
	st, capture := newQueryTest(t)
	res, err := st.Backfill002(context.Background())
	if err != nil {
		t.Fatalf("Backfill002: %v", err)
	}
	if res.Tenants != 0 || res.Documents != 0 {
		t.Errorf("empty scan should touch nothing, got %+v", res)
	}

	q := lastQuery(t, capture)
	wantSQL(t, q,
		`FROM "documents"`,
		`"app_id" IS NULL`,
		`"id" > ?`,
		`ORDER BY "id" ASC`,
		`LIMIT 500`,
	)
}
