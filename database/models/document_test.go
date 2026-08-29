package models

import (
	"testing"

	"github.com/lib/pq"
)

func TestDocument_GetID(t *testing.T) {
	if (Document{ID: "d1"}).GetID() != "d1" {
		t.Error("GetID mismatch")
	}
}

func TestDocument_Clone(t *testing.T) {
	coll := "c1"
	orig := Document{
		ID:           "d1",
		Key:          "a.md",
		AppID:        "a1",
		CollectionID: &coll,
		Name:         "install.md",
		Tags:         pq.StringArray{"docs", "guide"},
	}
	c := orig.Clone()

	// Deep copies: mutating the clone must not touch the original.
	*c.CollectionID = "c2"
	if *orig.CollectionID != "c1" {
		t.Error("Clone shares CollectionID pointer")
	}
	c.Tags[0] = "changed"
	if orig.Tags[0] != "docs" {
		t.Error("Clone shares Tags backing array")
	}
}

func TestDocument_Clone_NilFields(t *testing.T) {
	c := Document{ID: "d2"}.Clone()
	if c.Tags != nil || c.CollectionID != nil || c.DeletedAt != nil {
		t.Error("Clone should leave nil fields nil")
	}
}
