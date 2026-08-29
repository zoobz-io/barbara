package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// Document lifecycle statuses, surfaced on document responses. Derived from
// the published pointer alone — no version lookup needed.
const (
	statusDraft     = "draft"
	statusPublished = "published"
)

// documentStatus is draft when the document has no published pointer, published
// otherwise.
func documentStatus(d *models.Document) string {
	if d.PublishedVersionID == nil {
		return statusDraft
	}
	return statusPublished
}

// DocumentToResponse maps a document model to its authoring response, including
// lifecycle status.
func DocumentToResponse(d *models.Document) wire.DocumentResponse {
	return wire.DocumentResponse{
		ID:                 d.ID,
		TenantID:           d.TenantID,
		Key:                d.Key,
		PublishedVersionID: d.PublishedVersionID,
		Status:             documentStatus(d),
		Tags:               []string(d.Tags),
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// DocumentsToListResponse maps a slice of documents to the authoring list
// response, each carrying lifecycle status.
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

// DocumentContentToResponse maps a document and its head version to the editing
// read: the document plus a content block, or a null content block when the
// document has no versions yet.
func DocumentContentToResponse(dh *models.DocumentHead) wire.DocumentContentResponse {
	resp := wire.DocumentContentResponse{Document: DocumentToResponse(dh.Document)}
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
