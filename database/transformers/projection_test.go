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

	idx := Projection(doc, version)

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
