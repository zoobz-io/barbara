package models

import "testing"

func TestRelease_GetID(t *testing.T) {
	if (Release{ID: "r1"}).GetID() != "r1" {
		t.Error("GetID mismatch")
	}
}

func TestRelease_Clone(t *testing.T) {
	label, src, subj := "initial site", "r0", "d1"
	orig := Release{ID: "r1", AppID: "a1", Number: 3, CreatedBy: "u1", Kind: ReleaseKindRollback,
		Label: &label, SourceReleaseID: &src, SubjectDocumentID: &subj, Added: 1}
	c := orig.Clone()
	c.Number = 4
	*c.Label = "changed"
	*c.SourceReleaseID = "changed"
	*c.SubjectDocumentID = "changed"
	if orig.Number != 3 || *orig.Label != "initial site" || *orig.SourceReleaseID != "r0" || *orig.SubjectDocumentID != "d1" {
		t.Error("Clone is not independent of the original")
	}
	if n := (Release{}).Clone(); n.Label != nil || n.SourceReleaseID != nil || n.SubjectDocumentID != nil {
		t.Error("Clone of nil optionals should stay nil")
	}
}

func TestReleaseEntry_GetID(t *testing.T) {
	if (ReleaseEntry{ID: "e1"}).GetID() != "e1" {
		t.Error("GetID mismatch")
	}
}

func TestReleaseEntry_Clone(t *testing.T) {
	orig := ReleaseEntry{ID: "e1", ReleaseID: "r1", Key: "guides/install.md", DocumentID: "d1", VersionID: "v1", VersionNumber: 2}
	c := orig.Clone()
	c.Key = "changed"
	c.VersionNumber = 9
	if orig.Key != "guides/install.md" || orig.VersionNumber != 2 {
		t.Error("Clone is not independent of the original")
	}
}

func TestReleaseChange_GetID(t *testing.T) {
	if (ReleaseChange{ID: "c1"}).GetID() != "c1" {
		t.Error("GetID mismatch")
	}
}

func TestReleaseChange_Clone(t *testing.T) {
	prevKey, prevV, newV := "old.md", "v1", "v2"
	prevN, newN := 1, 2
	orig := ReleaseChange{ID: "c1", ReleaseID: "r1", Key: "new.md", DocumentID: "d1", Change: ChangeMoved,
		PrevKey: &prevKey, PrevVersionID: &prevV, PrevVersionNumber: &prevN, VersionID: &newV, VersionNumber: &newN}
	c := orig.Clone()
	*c.PrevKey = "x"
	*c.PrevVersionID = "x"
	*c.PrevVersionNumber = 9
	*c.VersionID = "x"
	*c.VersionNumber = 9
	if *orig.PrevKey != "old.md" || *orig.PrevVersionID != "v1" || *orig.PrevVersionNumber != 1 ||
		*orig.VersionID != "v2" || *orig.VersionNumber != 2 {
		t.Error("Clone is not independent of the original")
	}
	if n := (ReleaseChange{}).Clone(); n.PrevKey != nil || n.VersionNumber != nil {
		t.Error("Clone of nil optionals should stay nil")
	}
}
