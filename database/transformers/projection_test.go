package transformers

import (
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/zoobz-io/barbara/database/models"
)

func TestProjection(t *testing.T) {
	now := time.Now()
	doc := &models.Document{
		ID:        "d1",
		TenantID:  "t1",
		Key:       "guides/a.md",
		Tags:      pq.StringArray{"docs", "guide"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	version := &models.Version{ID: "v1", VersionNumber: 3, Content: "# hello"}

	idx := Projection(doc, version, doc.Key)

	if idx.DocumentID != "d1" || idx.TenantID != "t1" || idx.Key != "guides/a.md" {
		t.Errorf("document metadata not merged: %+v", idx)
	}
	if idx.VersionID != "v1" || idx.VersionNumber != 3 || idx.Content != "# hello" {
		t.Errorf("version fields not merged: %+v", idx)
	}
	if len(idx.Tags) != 2 || idx.Tags[0] != "docs" {
		t.Errorf("tags not carried: %v", idx.Tags)
	}
}

// app_id and parent_path are materialized: parent_path is the key's folder, and
// a nil app_id projects as empty.
func TestProjection_MaterializesAppAndParentPath(t *testing.T) {
	app := "app-1"
	version := &models.Version{ID: "v1"}

	nested := Projection(&models.Document{ID: "d1", Key: "guides/api/ref.md", AppID: &app}, version, "guides/api/ref.md")
	if nested.AppID != "app-1" || nested.ParentPath != "guides/api" {
		t.Errorf("nested projection = app:%q parent:%q; want app-1/guides/api", nested.AppID, nested.ParentPath)
	}

	root := Projection(&models.Document{ID: "d2", Key: "readme.md"}, version, "readme.md")
	if root.AppID != "" || root.ParentPath != "" {
		t.Errorf("root projection = app:%q parent:%q; want empty/empty", root.AppID, root.ParentPath)
	}
}

// parentPath derives the folder from the materialized key: everything before
// the last slash, "" at the root. The empty-string root convention must match
// what the folder read resolves an absent path parameter to.
func TestParentPath(t *testing.T) {
	cases := map[string]string{
		"readme.md":         "",
		"guides/install.md": "guides",
		"a/b/c.md":          "a/b",
		"/leading.md":       "",
	}
	for key, want := range cases {
		if got := parentPath(key); got != want {
			t.Errorf("parentPath(%q) = %q, want %q", key, got, want)
		}
	}
}

// The serving key wins over the authoring key: a document renamed after a
// release is cut still serves at the path the release recorded.
func TestProjection_ServesTheEntryKey(t *testing.T) {
	app := "app-1"
	doc := &models.Document{ID: "d1", Key: "renamed/new.md", AppID: &app}
	idx := Projection(doc, &models.Version{ID: "v1"}, "old/original.md")
	if idx.Key != "old/original.md" || idx.ParentPath != "old" {
		t.Errorf("projection = key:%q parent:%q; want the release-recorded path old/original.md", idx.Key, idx.ParentPath)
	}
}
