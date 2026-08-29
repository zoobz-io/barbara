package models

import "testing"

func TestRelease_GetID(t *testing.T) {
	if (Release{ID: "r1"}).GetID() != "r1" {
		t.Error("GetID mismatch")
	}
}

func TestRelease_Clone(t *testing.T) {
	orig := Release{ID: "r1", AppID: "a1", Number: 3, CreatedBy: "u1"}
	c := orig.Clone()
	c.Number = 4
	if orig.Number != 3 {
		t.Error("Clone is not independent of the original")
	}
}

func TestReleaseEntry_GetID(t *testing.T) {
	if (ReleaseEntry{ID: "e1"}).GetID() != "e1" {
		t.Error("GetID mismatch")
	}
}

func TestReleaseEntry_Clone(t *testing.T) {
	orig := ReleaseEntry{ID: "e1", ReleaseID: "r1", Key: "guides/install.md", DocumentID: "d1", VersionID: "v1"}
	c := orig.Clone()
	c.Key = "changed"
	if orig.Key != "guides/install.md" {
		t.Error("Clone is not independent of the original")
	}
}
