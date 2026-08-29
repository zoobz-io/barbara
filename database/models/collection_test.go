package models

import "testing"

func TestCollection_GetID(t *testing.T) {
	if (Collection{ID: "c1"}).GetID() != "c1" {
		t.Error("GetID mismatch")
	}
}

func TestCollection_Clone(t *testing.T) {
	parent := "c0"
	orig := Collection{ID: "c1", AppID: "a1", Name: "guides", ParentID: &parent}
	c := orig.Clone()

	*c.ParentID = "c9"
	if *orig.ParentID != "c0" {
		t.Error("Clone shares ParentID pointer")
	}
}

func TestCollection_Clone_NilFields(t *testing.T) {
	if (Collection{ID: "c2"}).Clone().ParentID != nil {
		t.Error("Clone should leave nil ParentID nil")
	}
}
