package transformers

import (
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/zoobz-io/barbara/database/models"
)

func TestDocumentToResponse(t *testing.T) {
	now := time.Now()
	d := &models.Document{
		ID:        "d1",
		TenantID:  "t1",
		Key:       "a.md",
		Tags:      pq.StringArray{"docs"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	r := DocumentToResponse(d, models.StatusPublished)
	if r.ID != "d1" || r.Key != "a.md" || r.TenantID != "t1" {
		t.Errorf("unexpected response: %+v", r)
	}
	if r.Status != "published" {
		t.Errorf("status = %q, want published", r.Status)
	}
	if len(r.Tags) != 1 || r.Tags[0] != "docs" {
		t.Errorf("tags not mapped: %v", r.Tags)
	}
}

// Status is passed through from the store; the transformer no longer derives it.
func TestDocumentToResponse_Status(t *testing.T) {
	if got := DocumentToResponse(&models.Document{ID: "d"}, models.StatusDraft).Status; got != "draft" {
		t.Errorf("status = %q, want draft", got)
	}
	if got := DocumentToResponse(&models.Document{ID: "d"}, models.StatusPublishedWithNewerDraft).Status; got != "published-with-newer-draft" {
		t.Errorf("status = %q, want published-with-newer-draft", got)
	}
}

func TestDocumentsToListResponse(t *testing.T) {
	docs := []*models.Document{
		{ID: "d1", Key: "a.md"},
		{ID: "d2", Key: "b.md"},
	}
	statuses := map[string]string{"d1": models.StatusPublished, "d2": models.StatusDraft}
	r := DocumentsToListResponse(docs, statuses, 10, 5)
	if r.Total != 2 || r.Limit != 10 || r.Offset != 5 {
		t.Errorf("unexpected list meta: %+v", r)
	}
	if len(r.Documents) != 2 || r.Documents[1].ID != "d2" {
		t.Errorf("documents not mapped: %+v", r.Documents)
	}
	if r.Documents[0].Status != "published" || r.Documents[1].Status != "draft" {
		t.Errorf("list statuses = %q,%q; want published,draft", r.Documents[0].Status, r.Documents[1].Status)
	}
}

// The content read carries the head version; an empty document has a null block.
func TestDocumentContentToResponse(t *testing.T) {
	now := time.Now()
	withHead := &models.DocumentHead{
		Document: &models.Document{ID: "d1", Key: "a.md"},
		Head:     &models.Version{ID: "v3", VersionNumber: 3, Content: "# head", CreatedAt: now},
	}
	r := DocumentContentToResponse(withHead, models.StatusPublished)
	if r.Document.ID != "d1" || r.Document.Status != "published" {
		t.Errorf("document not mapped: %+v", r.Document)
	}
	if r.Content == nil {
		t.Fatal("content block is nil, want the head version")
	}
	if r.Content.VersionID != "v3" || r.Content.VersionNumber != 3 || r.Content.Content != "# head" {
		t.Errorf("content block = %+v, want the head version", r.Content)
	}

	// Empty document (no versions): null content block, not a 404.
	empty := &models.DocumentHead{Document: &models.Document{ID: "d2", Key: "empty.md"}, Head: nil}
	if got := DocumentContentToResponse(empty, models.StatusDraft); got.Content != nil {
		t.Errorf("empty document content = %+v, want nil", got.Content)
	}
}
