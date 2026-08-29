package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// AppToResponse maps an app model to its authoring response.
func AppToResponse(a *models.App) wire.AppResponse {
	return wire.AppResponse{
		ID:               a.ID,
		TenantID:         a.TenantID,
		Name:             a.Name,
		CurrentReleaseID: a.CurrentReleaseID,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

// AppsToListResponse maps a slice of apps to the authoring list response.
func AppsToListResponse(apps []*models.App, limit, offset int) wire.AppListResponse {
	out := wire.AppListResponse{
		Apps:   make([]wire.AppResponse, len(apps)),
		Total:  len(apps),
		Limit:  limit,
		Offset: offset,
	}
	for i, a := range apps {
		out.Apps[i] = AppToResponse(a)
	}
	return out
}
