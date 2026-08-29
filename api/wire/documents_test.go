package wire

import "testing"

func TestDocumentResponse_Clone(t *testing.T) {
	pv := "v1"
	orig := DocumentResponse{ID: "d1", PublishedVersionID: &pv, Tags: []string{"docs"}}
	c := orig.Clone()

	*c.PublishedVersionID = "v2"
	if *orig.PublishedVersionID != "v1" {
		t.Error("Clone shares PublishedVersionID pointer")
	}
	c.Tags[0] = "changed"
	if orig.Tags[0] != "docs" {
		t.Error("Clone shares Tags slice")
	}
}

func TestDocumentListResponse_Clone(t *testing.T) {
	orig := DocumentListResponse{
		Documents: []DocumentResponse{{ID: "d1", Tags: []string{"a"}}},
		Total:     1,
	}
	c := orig.Clone()
	c.Documents[0].Tags[0] = "changed"
	if orig.Documents[0].Tags[0] != "a" {
		t.Error("Clone shares nested document tags")
	}
}
