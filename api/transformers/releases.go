package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// ReleaseToResponse maps a release model to its authoring response.
func ReleaseToResponse(r *models.Release) wire.ReleaseResponse {
	return wire.ReleaseResponse{
		ID:        r.ID,
		AppID:     r.AppID,
		TenantID:  r.TenantID,
		Number:    r.Number,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
	}
}

// ReleasesToListResponse maps a slice of releases to the authoring list response.
func ReleasesToListResponse(releases []*models.Release, limit, offset int) wire.ReleaseListResponse {
	out := wire.ReleaseListResponse{
		Releases: make([]wire.ReleaseResponse, len(releases)),
		Total:    len(releases),
		Limit:    limit,
		Offset:   offset,
	}
	for i, r := range releases {
		out.Releases[i] = ReleaseToResponse(r)
	}
	return out
}

// ReleaseWithEntriesToResponse maps a release and its entries to the get response.
func ReleaseWithEntriesToResponse(release *models.Release, entries []*models.ReleaseEntry) wire.ReleaseWithEntriesResponse {
	out := wire.ReleaseWithEntriesResponse{
		Release: ReleaseToResponse(release),
		Entries: make([]wire.ReleaseEntryResponse, len(entries)),
	}
	for i, e := range entries {
		out.Entries[i] = wire.ReleaseEntryResponse{
			Key:        e.Key,
			DocumentID: e.DocumentID,
			VersionID:  e.VersionID,
		}
	}
	return out
}
