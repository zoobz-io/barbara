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
	pv, app, coll, name := "v1", "a1", "c1", "install.md"
	orig := Document{
		ID:                 "d1",
		Key:                "a.md",
		PublishedVersionID: &pv,
		AppID:              &app,
		CollectionID:       &coll,
		Name:               &name,
		Tags:               pq.StringArray{"docs", "guide"},
	}
	c := orig.Clone()

	// Deep copies: mutating the clone must not touch the original.
	*c.PublishedVersionID = "v2"
	if *orig.PublishedVersionID != "v1" {
		t.Error("Clone shares PublishedVersionID pointer")
	}
	*c.AppID = "a2"
	if *orig.AppID != "a1" {
		t.Error("Clone shares AppID pointer")
	}
	*c.CollectionID = "c2"
	if *orig.CollectionID != "c1" {
		t.Error("Clone shares CollectionID pointer")
	}
	*c.Name = "renamed.md"
	if *orig.Name != "install.md" {
		t.Error("Clone shares Name pointer")
	}
	c.Tags[0] = "changed"
	if orig.Tags[0] != "docs" {
		t.Error("Clone shares Tags backing array")
	}
}

func TestDocument_Clone_NilFields(t *testing.T) {
	c := Document{ID: "d2"}.Clone()
	if c.PublishedVersionID != nil || c.Tags != nil || c.AppID != nil ||
		c.CollectionID != nil || c.Name != nil || c.DeletedAt != nil {
		t.Error("Clone should leave nil fields nil")
	}
}
