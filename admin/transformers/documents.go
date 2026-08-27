// Package transformers maps domain models to admin API wire types. Pure
// functions, no side effects. Admin sees everything — audit fields included, no
// masking.
package transformers

import (
	"github.com/zoobz-io/barbara/admin/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// DocumentToResponse maps a document model to its admin response.
func DocumentToResponse(d *models.Document) wire.DocumentResponse {
	return wire.DocumentResponse{
		ID:                 d.ID,
		TenantID:           d.TenantID,
		Key:                d.Key,
		PublishedVersionID: d.PublishedVersionID,
		Tags:               []string(d.Tags),
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// DocumentsToListResponse maps a slice of documents to the admin list response.
func DocumentsToListResponse(docs []*models.Document, limit, offset int) wire.DocumentListResponse {
	out := wire.DocumentListResponse{
		Documents: make([]wire.DocumentResponse, len(docs)),
		Total:     len(docs),
		Limit:     limit,
		Offset:    offset,
	}
	for i, d := range docs {
		out.Documents[i] = DocumentToResponse(d)
	}
	return out
}
