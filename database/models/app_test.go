package models

import "testing"

func TestApp_GetID(t *testing.T) {
	if (App{ID: "a1"}).GetID() != "a1" {
		t.Error("GetID mismatch")
	}
}

func TestApp_Clone(t *testing.T) {
	rel := "r1"
	orig := App{ID: "a1", Name: "docs", CurrentReleaseID: &rel}
	c := orig.Clone()

	*c.CurrentReleaseID = "r2"
	if *orig.CurrentReleaseID != "r1" {
		t.Error("Clone shares CurrentReleaseID pointer")
	}
}

func TestApp_Clone_NilFields(t *testing.T) {
	if (App{ID: "a2"}).Clone().CurrentReleaseID != nil {
		t.Error("Clone should leave nil CurrentReleaseID nil")
	}
}
