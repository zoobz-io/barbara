package transformers

import (
	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// ReleaseToResponse maps a release model to its authoring response.
func ReleaseToResponse(r *models.Release) wire.ReleaseResponse {
	return wire.ReleaseResponse{
		ID:                r.ID,
		AppID:             r.AppID,
		TenantID:          r.TenantID,
		Number:            r.Number,
		Kind:              r.Kind,
		Label:             r.Label,
		SourceReleaseID:   r.SourceReleaseID,
		SubjectDocumentID: r.SubjectDocumentID,
		EntryCount:        r.EntryCount,
		Added:             r.Added,
		Changed:           r.Changed,
		Removed:           r.Removed,
		Moved:             r.Moved,
		CreatedBy:         r.CreatedBy,
		CreatedAt:         r.CreatedAt,
	}.Clone()
}

// ReleasesToListResponse maps a page of releases and the app's total to the
// authoring list response.
func ReleasesToListResponse(releases []*models.Release, total int64, limit, offset int) wire.ReleaseListResponse {
	out := wire.ReleaseListResponse{
		Releases: make([]wire.ReleaseResponse, len(releases)),
		Total:    int(total),
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
			Key:           e.Key,
			DocumentID:    e.DocumentID,
			VersionID:     e.VersionID,
			VersionNumber: e.VersionNumber,
		}
	}
	return out
}

// ReleaseChangesToResponse maps a release and its changes to the changes response.
func ReleaseChangesToResponse(release *models.Release, changes []*models.ReleaseChange) wire.ReleaseChangesResponse {
	out := wire.ReleaseChangesResponse{
		Release: ReleaseToResponse(release),
		Changes: make([]wire.ReleaseChangeResponse, len(changes)),
	}
	for i, c := range changes {
		out.Changes[i] = wire.ReleaseChangeResponse{
			Key:               c.Key,
			DocumentID:        c.DocumentID,
			Change:            c.Change,
			PrevKey:           c.PrevKey,
			PrevVersionID:     c.PrevVersionID,
			PrevVersionNumber: c.PrevVersionNumber,
			VersionID:         c.VersionID,
			VersionNumber:     c.VersionNumber,
		}.Clone()
	}
	return out
}
