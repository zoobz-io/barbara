package wire

import "testing"

func TestPublishedDocumentResponse_Clone(t *testing.T) {
	orig := PublishedDocumentResponse{DocumentID: "d1", Tags: []string{"guide"}}
	c := orig.Clone()

	c.Tags[0] = "changed"
	if orig.Tags[0] != "guide" {
		t.Error("Clone shares Tags slice")
	}
}

func TestPublishedDocumentListResponse_Clone(t *testing.T) {
	orig := PublishedDocumentListResponse{
		Documents: []PublishedDocumentResponse{{DocumentID: "d1", Tags: []string{"a"}}},
		Total:     1,
	}
	c := orig.Clone()

	c.Documents[0].Tags[0] = "changed"
	if orig.Documents[0].Tags[0] != "a" {
		t.Error("Clone shares nested document tags")
	}
}
