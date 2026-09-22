// Package transformers maps domain models to public-API wire types. Pure
// functions, no side effects. Published reads drop internal fields (tenant_id,
// version_id) — the wire type simply has no field for them; authoring responses
// expose full data, audit fields included.
package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/internal/mdast"
)

// Response formats for the published document lookup.
const (
	// FormatMarkdown returns the raw markdown in Content. It is the default.
	FormatMarkdown = "markdown"
	// FormatMdast returns the parsed tree in Body and frontmatter in Meta.
	FormatMdast = "mdast"
)

// IndexToResponse maps a document projection to its site-facing response in the
// requested format. For FormatMdast it parses the content into an mdast tree and
// returns the tree and the frontmatter; for any other format it returns the raw
// markdown. URLs in the tree are returned exactly as authored — resolving them
// to fetchable locations is the site's job until the published URL layout is
// settled.
func IndexToResponse(d *models.DocumentIndex, format string) (wire.PublishedDocumentResponse, error) {
	if format == FormatMdast {
		return mdastResponse(d)
	}
	return markdownResponse(d), nil
}

// markdownResponse builds the raw-markdown response.
func markdownResponse(d *models.DocumentIndex) wire.PublishedDocumentResponse {
	return wire.PublishedDocumentResponse{
		DocumentID:    d.DocumentID,
		Key:           d.Key,
		Content:       d.Content,
		Tags:          d.Tags,
		VersionNumber: d.VersionNumber,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// mdastResponse builds the mdast response: the parsed tree and the frontmatter
// map.
func mdastResponse(d *models.DocumentIndex) (wire.PublishedDocumentResponse, error) {
	root, meta, err := mdast.Parse([]byte(d.Content))
	if err != nil {
		return wire.PublishedDocumentResponse{}, err
	}
	return wire.PublishedDocumentResponse{
		DocumentID:    d.DocumentID,
		Key:           d.Key,
		Body:          root,
		Meta:          meta,
		Tags:          d.Tags,
		VersionNumber: d.VersionNumber,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}, nil
}

// IndexesToListResponse maps a page of projections plus its total to the
// site-facing list response.
func IndexesToListResponse(docs []models.DocumentIndex, total int64, limit, offset int) wire.PublishedDocumentListResponse {
	out := wire.PublishedDocumentListResponse{
		Documents: make([]wire.PublishedDocumentResponse, len(docs)),
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}
	for i := range docs {
		out.Documents[i] = markdownResponse(&docs[i])
	}
	return out
}
