package wire

import "testing"

func TestVersionListResponse_Clone(t *testing.T) {
	orig := VersionListResponse{
		Versions: []VersionResponse{{ID: "v1"}, {ID: "v2"}},
		Total:    2,
	}
	c := orig.Clone()
	c.Versions[0].ID = "changed"
	if orig.Versions[0].ID != "v1" {
		t.Error("Clone shares the Versions slice")
	}
}
