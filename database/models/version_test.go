package models

import "testing"

func TestVersion_GetID(t *testing.T) {
	if (Version{ID: "v1"}).GetID() != "v1" {
		t.Error("GetID mismatch")
	}
}

func TestVersion_Clone(t *testing.T) {
	orig := Version{ID: "v1", DocumentID: "d1", VersionNumber: 2, Content: "hi", CreatedBy: "u1"}
	c := orig.Clone()
	if c != orig {
		t.Errorf("clone = %+v, want equal to %+v", c, orig)
	}
}
