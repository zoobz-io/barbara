// Package transformers maps domain models to public-API wire types. Pure
// functions, no side effects. Published reads drop internal fields (tenant_id,
// version_id) — the wire type simply has no field for them; authoring responses
// expose full data, audit fields included.
package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// IndexToResponse maps a document projection to its site-facing response.
func IndexToResponse(d *models.DocumentIndex) wire.PublishedDocumentResponse {
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
		out.Documents[i] = IndexToResponse(&docs[i])
	}
	return out
}
