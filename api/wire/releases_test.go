package wire

import (
	"strings"
	"testing"
)

func TestCutReleaseRequest_Validate(t *testing.T) {
	if err := (CutReleaseRequest{}).Validate(); err != nil {
		t.Errorf("empty label should validate, got %v", err)
	}
	if err := (CutReleaseRequest{Label: strings.Repeat("é", MaxReleaseLabelLen)}).Validate(); err != nil {
		t.Errorf("label at the bound (in runes) should validate, got %v", err)
	}
	if err := (CutReleaseRequest{Label: strings.Repeat("x", MaxReleaseLabelLen+1)}).Validate(); err == nil {
		t.Error("label past the bound should fail")
	}
}

func TestReleaseResponse_Clone(t *testing.T) {
	label, src := "launch", "r0"
	orig := ReleaseResponse{ID: "r1", Label: &label, SourceReleaseID: &src}
	c := orig.Clone()
	*c.Label = "changed"
	*c.SourceReleaseID = "changed"
	if *orig.Label != "launch" || *orig.SourceReleaseID != "r0" {
		t.Error("Clone shares optional fields")
	}
	if n := (ReleaseResponse{}).Clone(); n.Label != nil || n.SubjectDocumentID != nil {
		t.Error("Clone of nil optionals should stay nil")
	}
}

func TestReleaseWithEntriesResponse_Clone(t *testing.T) {
	label := "l"
	orig := ReleaseWithEntriesResponse{
		Release: ReleaseResponse{Label: &label},
		Entries: []ReleaseEntryResponse{{Key: "a.md", VersionNumber: 1}},
	}
	c := orig.Clone()
	c.Entries[0].Key = "changed"
	*c.Release.Label = "changed"
	if orig.Entries[0].Key != "a.md" || *orig.Release.Label != "l" {
		t.Error("Clone shares entries or the release's label")
	}
}

func TestReleaseChangesResponse_Clone(t *testing.T) {
	prevKey, n := "old.md", 2
	orig := ReleaseChangesResponse{
		Changes: []ReleaseChangeResponse{{Key: "new.md", Change: "moved", PrevKey: &prevKey, VersionNumber: &n}},
	}
	c := orig.Clone()
	*c.Changes[0].PrevKey = "changed"
	*c.Changes[0].VersionNumber = 9
	c.Changes[0].Key = "changed"
	if *orig.Changes[0].PrevKey != "old.md" || *orig.Changes[0].VersionNumber != 2 || orig.Changes[0].Key != "new.md" {
		t.Error("Clone shares change rows")
	}
	if empty := (ReleaseChangesResponse{}).Clone(); empty.Changes != nil {
		t.Error("Clone of nil changes should stay nil")
	}
}

func TestReleaseListResponse_Clone(t *testing.T) {
	label := "l"
	orig := ReleaseListResponse{Releases: []ReleaseResponse{{ID: "r1", Label: &label}}, Total: 1}
	c := orig.Clone()
	c.Releases[0].ID = "changed"
	*c.Releases[0].Label = "changed"
	if orig.Releases[0].ID != "r1" || *orig.Releases[0].Label != "l" {
		t.Error("Clone shares releases")
	}
}
