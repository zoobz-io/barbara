package transformers

import (
	"testing"

	"github.com/zoobz-io/barbara/database/models"
)

func TestVersionToResponse(t *testing.T) {
	v := &models.Version{ID: "v1", DocumentID: "d1", TenantID: "t1", Content: "hi", CreatedBy: "u1", VersionNumber: 3}
	r := VersionToResponse(v)
	if r.ID != "v1" || r.DocumentID != "d1" || r.CreatedBy != "u1" || r.VersionNumber != 3 || r.Content != "hi" {
		t.Errorf("unexpected response: %+v", r)
	}
}

func TestVersionsToListResponse(t *testing.T) {
	versions := []*models.Version{{ID: "v2", VersionNumber: 2}, {ID: "v1", VersionNumber: 1}}
	r := VersionsToListResponse(versions, 10, 5)
	if r.Total != 2 || r.Limit != 10 || r.Offset != 5 {
		t.Errorf("unexpected list meta: %+v", r)
	}
	if len(r.Versions) != 2 || r.Versions[0].ID != "v2" {
		t.Errorf("versions not mapped: %+v", r.Versions)
	}
}
