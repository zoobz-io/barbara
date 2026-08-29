package wire

import "testing"

// Clone deep-copies a collection response: mutating the copy's ParentID must not
// touch the original.
func TestCollectionResponse_Clone(t *testing.T) {
	parent := "p-1"
	orig := CollectionResponse{ID: "c-1", AppID: "app-1", Name: "guides", ParentID: &parent}
	cp := orig.Clone()
	if cp.ParentID == orig.ParentID {
		t.Fatal("Clone shares the ParentID pointer")
	}
	*cp.ParentID = "p-2"
	if *orig.ParentID != "p-1" {
		t.Errorf("mutating the copy changed the original: %q", *orig.ParentID)
	}

	// A nil ParentID clones to nil.
	if root := (CollectionResponse{ID: "c-2"}).Clone(); root.ParentID != nil {
		t.Errorf("root collection clone got a non-nil parent: %v", root.ParentID)
	}
}

// Clone deep-copies contents: the subcollection and document slices are
// independent of the original's.
func TestCollectionContentsResponse_Clone(t *testing.T) {
	orig := CollectionContentsResponse{
		Subcollections: []CollectionResponse{{ID: "c-1", Name: "guides"}},
		Documents:      []DocumentResponse{{ID: "d-1", Key: "a.md"}},
	}
	cp := orig.Clone()
	cp.Subcollections[0].Name = "changed"
	cp.Documents[0].Key = "b.md"
	if orig.Subcollections[0].Name != "guides" || orig.Documents[0].Key != "a.md" {
		t.Errorf("Clone did not isolate the slices: %+v", orig)
	}

	// Empty contents clone to empty (nil slices stay nil).
	empty := CollectionContentsResponse{}.Clone()
	if empty.Subcollections != nil || empty.Documents != nil {
		t.Errorf("empty contents clone got non-nil slices: %+v", empty)
	}
}
