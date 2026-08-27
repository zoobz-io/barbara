package transformers

import (
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/zoobz-io/barbara/database/models"
)

func TestDocumentToResponse(t *testing.T) {
	pv := "v1"
	now := time.Now()
	d := &models.Document{
		ID:                 "d1",
		TenantID:           "t1",
		Key:                "a.md",
		PublishedVersionID: &pv,
		Tags:               pq.StringArray{"docs"},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	r := DocumentToResponse(d)
	if r.ID != "d1" || r.Key != "a.md" || r.TenantID != "t1" {
		t.Errorf("unexpected response: %+v", r)
	}
	if r.PublishedVersionID == nil || *r.PublishedVersionID != "v1" {
		t.Error("published version not mapped")
	}
	if len(r.Tags) != 1 || r.Tags[0] != "docs" {
		t.Errorf("tags not mapped: %v", r.Tags)
	}
}

func TestDocumentsToListResponse(t *testing.T) {
	docs := []*models.Document{
		{ID: "d1", Key: "a.md"},
		{ID: "d2", Key: "b.md"},
	}
	r := DocumentsToListResponse(docs, 10, 5)
	if r.Total != 2 || r.Limit != 10 || r.Offset != 5 {
		t.Errorf("unexpected list meta: %+v", r)
	}
	if len(r.Documents) != 2 || r.Documents[1].ID != "d2" {
		t.Errorf("documents not mapped: %+v", r.Documents)
	}
}
