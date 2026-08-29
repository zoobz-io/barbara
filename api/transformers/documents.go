package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// DocumentToResponse maps a document model to its authoring response. Lifecycle
// status is derived from the app's current release, so it is computed by the
// store and passed in — the transformer no longer reads it off the row.
func DocumentToResponse(d *models.Document, status string) wire.DocumentResponse {
	return wire.DocumentResponse{
		ID:        d.ID,
		TenantID:  d.TenantID,
		Key:       d.Key,
		Status:    status,
		Tags:      []string(d.Tags),
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// DocumentsToListResponse maps documents to the authoring list response, each
// carrying its status (keyed by document id in statuses).
func DocumentsToListResponse(docs []*models.Document, statuses map[string]string, limit, offset int) wire.DocumentListResponse {
	out := wire.DocumentListResponse{
		Documents: make([]wire.DocumentResponse, len(docs)),
		Total:     len(docs),
		Limit:     limit,
		Offset:    offset,
	}
	for i, d := range docs {
		out.Documents[i] = DocumentToResponse(d, statuses[d.ID])
	}
	return out
}

// DocumentContentToResponse maps a document and its head version to the editing
// read: the document plus a content block, or a null content block when the
// document has no versions yet.
func DocumentContentToResponse(dh *models.DocumentHead, status string) wire.DocumentContentResponse {
	resp := wire.DocumentContentResponse{Document: DocumentToResponse(dh.Document, status)}
	if dh.Head != nil {
		resp.Content = &wire.ContentBlock{
			VersionID:     dh.Head.ID,
			VersionNumber: dh.Head.VersionNumber,
			Content:       dh.Head.Content,
			CreatedAt:     dh.Head.CreatedAt,
		}
	}
	return resp
}
