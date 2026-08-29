package transformers

import (
	"testing"

	"github.com/zoobz-io/barbara/database/models"
)

func TestCollectionToResponse(t *testing.T) {
	parent := "p-1"
	c := &models.Collection{ID: "c-1", TenantID: "t", AppID: "app-1", ParentID: &parent, Name: "guides"}
	r := CollectionToResponse(c)
	if r.ID != "c-1" || r.AppID != "app-1" || r.Name != "guides" || r.ParentID == nil || *r.ParentID != "p-1" {
		t.Errorf("unexpected response: %+v", r)
	}
}

// Contents maps subcollections and documents together, each document carrying
// its derived status.
func TestCollectionContentsToResponse(t *testing.T) {
	published := "v-1"
	cc := &models.CollectionContents{
		Subcollections: []*models.Collection{{ID: "c-2", Name: "sub"}},
		Documents: []*models.Document{
			{ID: "d-1", Key: "a.md"},                              // draft
			{ID: "d-2", Key: "b.md", PublishedVersionID: &published}, // published
		},
	}
	r := CollectionContentsToResponse(cc)
	if len(r.Subcollections) != 1 || len(r.Documents) != 2 {
		t.Fatalf("unexpected shape: %+v", r)
	}
	if r.Documents[0].Status != statusDraft || r.Documents[1].Status != statusPublished {
		t.Errorf("document statuses = %q, %q", r.Documents[0].Status, r.Documents[1].Status)
	}
}
