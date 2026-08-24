package models

import "testing"

func TestDocumentIndex_Clone(t *testing.T) {
	d := DocumentIndex{
		DocumentID:    "doc-1",
		VersionID:     "ver-1",
		TenantID:      "tenant-1",
		Key:           "guides/install.md",
		Content:       "# Install",
		VersionNumber: 3,
		Tags:          []string{"guide", "public"},
	}

	clone := d.Clone()

	if clone.DocumentID != d.DocumentID || clone.Key != d.Key || clone.VersionNumber != d.VersionNumber {
		t.Error("scalar fields not copied")
	}

	// Slice independence — mutating the clone must not touch the original.
	clone.Tags[0] = "mutated"
	if d.Tags[0] != "guide" {
		t.Error("Tags mutation leaked to original")
	}
}

func TestDocumentIndex_Clone_NilTags(t *testing.T) {
	d := DocumentIndex{DocumentID: "doc-1"}
	clone := d.Clone()
	if clone.Tags != nil {
		t.Error("expected nil Tags to stay nil")
	}
}
