package transformers

import (
	"encoding/json"
	"strings"
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

	resp, err := IndexToResponse(idx, FormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}

	if resp.DocumentID != "d1" || resp.Key != "guides/install.md" || resp.Content != "how to install" {
		t.Errorf("public fields not carried: %+v", resp)
	}
	if resp.VersionNumber != 3 || len(resp.Tags) != 2 {
		t.Errorf("metadata not carried: %+v", resp)
	}
	if resp.Body != nil || resp.Meta != nil {
		t.Errorf("markdown format must not set body/meta: %+v", resp)
	}
	// The wire type structurally has no tenant_id/version_id field — the marshaled
	// response can never leak them. Clone must be independent of the source tags.
	c := resp.Clone()
	c.Tags[0] = "mutated"
	if resp.Tags[0] == "mutated" {
		t.Error("Clone did not deep-copy tags")
	}
}

func TestIndexToResponse_Mdast(t *testing.T) {
	idx := &models.DocumentIndex{
		DocumentID: "d1",
		Key:        "guides/install.md",
		ParentPath: "guides",
		Content:    "---\ntitle: Install\n---\n\n# Install\n\n![diagram](diagram.png)\n",
	}

	resp, err := IndexToResponse(idx, FormatMdast)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "" {
		t.Errorf("mdast format must omit content, got %q", resp.Content)
	}
	if resp.Body == nil {
		t.Fatal("mdast format must set body")
	}
	if resp.Meta["title"] != "Install" {
		t.Errorf("frontmatter not returned: meta = %v", resp.Meta)
	}
	body, err := json.Marshal(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	// URLs come back exactly as authored; nothing resolves them here.
	const want = `"url":"diagram.png"`
	if !strings.Contains(string(body), want) {
		t.Errorf("image URL not returned as authored; body = %s", body)
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
