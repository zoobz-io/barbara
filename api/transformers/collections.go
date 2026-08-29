package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// CollectionToResponse maps a collection model to its authoring response.
func CollectionToResponse(c *models.Collection) wire.CollectionResponse {
	return wire.CollectionResponse{
		ID:        c.ID,
		TenantID:  c.TenantID,
		AppID:     c.AppID,
		ParentID:  c.ParentID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// CollectionContentsToResponse maps a collection's contents to the one-round-trip
// listing: subcollections by name, documents by key each with derived status.
func CollectionContentsToResponse(cc *models.CollectionContents) wire.CollectionContentsResponse {
	out := wire.CollectionContentsResponse{
		Subcollections: make([]wire.CollectionResponse, len(cc.Subcollections)),
		Documents:      make([]wire.DocumentResponse, len(cc.Documents)),
	}
	for i, c := range cc.Subcollections {
		out.Subcollections[i] = CollectionToResponse(c)
	}
	for i, d := range cc.Documents {
		out.Documents[i] = DocumentToResponse(d)
	}
	return out
}
