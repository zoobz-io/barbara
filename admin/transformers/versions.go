package transformers

import (
	"github.com/zoobz-io/barbara/admin/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// VersionToResponse maps a version model to its admin response.
func VersionToResponse(v *models.Version) wire.VersionResponse {
	return wire.VersionResponse{
		ID:            v.ID,
		DocumentID:    v.DocumentID,
		TenantID:      v.TenantID,
		Content:       v.Content,
		CreatedBy:     v.CreatedBy,
		VersionNumber: v.VersionNumber,
		CreatedAt:     v.CreatedAt,
	}
}

// VersionsToListResponse maps a slice of versions to the admin list response.
func VersionsToListResponse(versions []*models.Version, limit, offset int) wire.VersionListResponse {
	out := wire.VersionListResponse{
		Versions: make([]wire.VersionResponse, len(versions)),
		Total:    len(versions),
		Limit:    limit,
		Offset:   offset,
	}
	for i, v := range versions {
		out.Versions[i] = VersionToResponse(v)
	}
	return out
}
