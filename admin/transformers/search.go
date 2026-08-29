// Package transformers maps domain models to admin (internal-platform) wire
// types. Pure functions, no side effects.
package transformers

import (
	"github.com/zoobz-io/barbara/admin/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// SearchResultToResponse maps a document projection to an admin search hit,
// including the owning tenant.
func SearchResultToResponse(d *models.DocumentIndex) wire.SearchResultResponse {
	return wire.SearchResultResponse{
		DocumentID:    d.DocumentID,
		TenantID:      d.TenantID,
		Key:           d.Key,
		Content:       d.Content,
		Tags:          d.Tags,
		VersionNumber: d.VersionNumber,
		CreatedAt:     d.CreatedAt,
	}
}

// SearchResultsToResponse maps a page of projections plus its total to the admin
// search response.
func SearchResultsToResponse(docs []models.DocumentIndex, total int64, limit, offset int) wire.SearchResultsResponse {
	out := wire.SearchResultsResponse{
		Results: make([]wire.SearchResultResponse, len(docs)),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}
	for i := range docs {
		out.Results[i] = SearchResultToResponse(&docs[i])
	}
	return out
}
