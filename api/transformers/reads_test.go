package transformers

import (
	"testing"
	"time"

	"github.com/zoobz-io/barbara/database/models"
)

func TestIndexToResponse_DropsInternalFields(t *testing.T) {
	now := time.Now()
	idx := &models.DocumentIndex{
		DocumentID:    "d1",
		TenantID:      "t1",
		VersionID:     "v1",
		Key:           "guides/install.md",
		Content:       "how to install",
		Tags:          []string{"guide", "setup"},
		VersionNumber: 3,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := IndexToResponse(idx)

	if resp.DocumentID != "d1" || resp.Key != "guides/install.md" || resp.Content != "how to install" {
		t.Errorf("public fields not carried: %+v", resp)
	}
	if resp.VersionNumber != 3 || len(resp.Tags) != 2 {
		t.Errorf("metadata not carried: %+v", resp)
	}
	// The wire type structurally has no tenant_id/version_id field — the marshaled
	// response can never leak them. Clone must be independent of the source tags.
	c := resp.Clone()
	c.Tags[0] = "mutated"
	if resp.Tags[0] == "mutated" {
		t.Error("Clone did not deep-copy tags")
	}
}

func TestIndexesToListResponse(t *testing.T) {
	docs := []models.DocumentIndex{
		{DocumentID: "d1", Key: "a.md"},
		{DocumentID: "d2", Key: "b.md"},
	}
	out := IndexesToListResponse(docs, 17, 50, 0)

	if out.Total != 17 {
		t.Errorf("total = %d, want 17 (the full match count, not the page size)", out.Total)
	}
	if len(out.Documents) != 2 || out.Documents[1].DocumentID != "d2" {
		t.Errorf("page not carried: %+v", out.Documents)
	}
	if out.Limit != 50 || out.Offset != 0 {
		t.Errorf("pagination not carried: %+v", out)
	}
}

func TestPublishedAssetResolver(t *testing.T) {
	cases := []struct {
		name       string
		parentPath string
		in         string
		want       string
	}{
		{"absolute scheme", "guides", "https://cdn.example.com/x.png", "https://cdn.example.com/x.png"},
		{"mailto scheme", "guides", "mailto:team@example.com", "mailto:team@example.com"},
		{"rooted path", "guides", "/static/x.png", "/static/x.png"},
		{"fragment", "guides", "#section", "#section"},
		{"relative in folder", "guides/api", "diagram.png", "/published/apps/app-1/assets/object?key=guides/api/diagram.png"},
		{"dot-dot segments", "guides/api", "../assets/diagram.png", "/published/apps/app-1/assets/object?key=guides/assets/diagram.png"},
		{"document at app root", "", "logo.png", "/published/apps/app-1/assets/object?key=logo.png"},
		{"relative with space", "guides", "a b.png", "/published/apps/app-1/assets/object?key=guides/a+b.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resolve := PublishedAssetResolver("app-1", c.parentPath)
			if got := resolve(c.in); got != c.want {
				t.Errorf("resolve(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
