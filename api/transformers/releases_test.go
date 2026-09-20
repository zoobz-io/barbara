package transformers

import (
	"testing"
	"time"

	"github.com/zoobz-io/barbara/database/models"
)

func TestReleaseToResponse(t *testing.T) {
	label, src, subj := "launch", "r0", "d9"
	now := time.Now()
	r := &models.Release{
		ID: "r1", AppID: "a1", TenantID: "t1", Number: 4, CreatedBy: "u1", CreatedAt: now,
		Kind: models.ReleaseKindRollback, Label: &label, SourceReleaseID: &src, SubjectDocumentID: &subj,
		EntryCount: 10, Added: 1, Changed: 2, Removed: 3, Moved: 4,
	}
	got := ReleaseToResponse(r)
	if got.ID != "r1" || got.AppID != "a1" || got.TenantID != "t1" || got.Number != 4 || got.CreatedBy != "u1" || !got.CreatedAt.Equal(now) {
		t.Errorf("identity fields = %+v", got)
	}
	if got.Kind != "rollback" || *got.Label != "launch" || *got.SourceReleaseID != "r0" || *got.SubjectDocumentID != "d9" {
		t.Errorf("provenance fields = %+v", got)
	}
	if got.EntryCount != 10 || got.Added != 1 || got.Changed != 2 || got.Removed != 3 || got.Moved != 4 {
		t.Errorf("counts = %+v", got)
	}
	*got.Label = "changed"
	if *r.Label != "launch" {
		t.Error("response shares the model's label")
	}
}

func TestReleasesToListResponse(t *testing.T) {
	got := ReleasesToListResponse([]*models.Release{{ID: "r2", Number: 2}, {ID: "r1", Number: 1}}, 12, 2, 0)
	if got.Total != 12 || got.Limit != 2 || got.Offset != 0 || len(got.Releases) != 2 || got.Releases[0].ID != "r2" {
		t.Errorf("list = %+v", got)
	}
	if empty := ReleasesToListResponse(nil, 0, 10, 0); empty.Releases == nil || len(empty.Releases) != 0 {
		t.Errorf("empty list should be an empty slice, got %+v", empty.Releases)
	}
}

func TestReleaseWithEntriesToResponse(t *testing.T) {
	got := ReleaseWithEntriesToResponse(&models.Release{ID: "r1"}, []*models.ReleaseEntry{
		{Key: "a.md", DocumentID: "d1", VersionID: "v1", VersionNumber: 3},
	})
	if got.Release.ID != "r1" || len(got.Entries) != 1 {
		t.Fatalf("response = %+v", got)
	}
	if e := got.Entries[0]; e.Key != "a.md" || e.DocumentID != "d1" || e.VersionID != "v1" || e.VersionNumber != 3 {
		t.Errorf("entry = %+v", e)
	}
}

func TestReleaseChangesToResponse(t *testing.T) {
	prevKey, prevID, newID := "old.md", "v1", "v2"
	prevN, newN := 1, 2
	got := ReleaseChangesToResponse(&models.Release{ID: "r2", Moved: 1}, []*models.ReleaseChange{
		{Key: "new.md", DocumentID: "d1", Change: models.ChangeMoved,
			PrevKey: &prevKey, PrevVersionID: &prevID, PrevVersionNumber: &prevN, VersionID: &newID, VersionNumber: &newN},
		{Key: "gone.md", DocumentID: "d2", Change: models.ChangeRemoved, PrevVersionID: &prevID, PrevVersionNumber: &prevN},
	})
	if got.Release.Moved != 1 || len(got.Changes) != 2 {
		t.Fatalf("response = %+v", got)
	}
	m := got.Changes[0]
	if m.Change != "moved" || *m.PrevKey != "old.md" || *m.PrevVersionID != "v1" || *m.PrevVersionNumber != 1 || *m.VersionID != "v2" || *m.VersionNumber != 2 {
		t.Errorf("moved row = %+v", m)
	}
	r := got.Changes[1]
	if r.Change != "removed" || r.VersionID != nil || r.VersionNumber != nil || r.PrevKey != nil || *r.PrevVersionNumber != 1 {
		t.Errorf("removed row = %+v", r)
	}
	if empty := ReleaseChangesToResponse(&models.Release{}, nil); empty.Changes == nil || len(empty.Changes) != 0 {
		t.Errorf("no changes should be an empty slice, got %+v", empty.Changes)
	}
}
