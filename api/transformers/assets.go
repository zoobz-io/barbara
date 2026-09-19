package transformers

import (
	"time"

	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// AssetToResponse maps an asset model to its authoring metadata response (no bytes).
func AssetToResponse(a *models.Asset) wire.AssetResponse {
	out := wire.AssetResponse{
		Key:         a.Key,
		ContentType: a.ContentType,
		Kind:        string(models.KindOf(a.ContentType)),
		Size:        a.Size,
	}
	if !a.LastModified.IsZero() {
		t := a.LastModified
		out.LastModified = &t
	}
	return out
}

// AssetsToListResponse maps a slice of assets to the authoring list response.
func AssetsToListResponse(assets []*models.Asset) wire.AssetListResponse {
	out := wire.AssetListResponse{
		Assets: make([]wire.AssetResponse, len(assets)),
		Total:  len(assets),
	}
	for i, a := range assets {
		out.Assets[i] = AssetToResponse(a)
	}
	return out
}

// AssetLevelToFolderResponse maps one level of the asset tree to its folder
// response. Empty levels serialize as empty arrays, never null.
func AssetLevelToFolderResponse(level *models.AssetLevel) wire.AssetFolderResponse {
	out := wire.AssetFolderResponse{
		Path:    level.Path,
		Folders: make([]wire.AssetSubfolderResponse, len(level.Folders)),
		Assets:  make([]wire.AssetResponse, len(level.Assets)),
	}
	for i, f := range level.Folders {
		out.Folders[i] = wire.AssetSubfolderResponse{Name: f.Name, Count: f.Count, Size: f.Size, LastWrittenAt: f.LastWrittenAt}
	}
	for i, a := range level.Assets {
		out.Assets[i] = AssetToResponse(a)
	}
	return out
}

// AssetStatsToResponse maps the bookkeeping view to its response. The day
// rows come flat — one per kind per day, with the all-kinds row keyed "" —
// and are regrouped by day, each day carrying its total and its kinds. Empty
// lists serialize as empty arrays, never null.
func AssetStatsToResponse(stats *models.AssetStats) wire.AssetStatsResponse {
	out := wire.AssetStatsResponse{
		ComputedAt:    stats.ComputedAt,
		LastWrittenAt: stats.Root.LastWrittenAt,
		Count:         stats.Root.Count,
		Size:          stats.Root.Bytes,
		Kinds:         make([]wire.AssetKindStatsResponse, 0, len(stats.Kinds)),
		Days:          []wire.AssetDayStatsResponse{},
	}
	for _, k := range stats.Kinds {
		out.Kinds = append(out.Kinds, wire.AssetKindStatsResponse{
			LastWrittenAt: k.LastWrittenAt, Kind: string(k.Kind), Count: k.Count, Size: k.Bytes,
		})
	}
	// Rows arrive ordered by day, so each day's rows are contiguous; a new
	// day opens a new entry.
	for _, d := range stats.Days {
		day := d.Day.UTC().Format(time.DateOnly)
		if n := len(out.Days); n == 0 || out.Days[n-1].Day != day {
			out.Days = append(out.Days, wire.AssetDayStatsResponse{Day: day, Kinds: []wire.AssetKindCountResponse{}})
		}
		cur := &out.Days[len(out.Days)-1]
		if d.Kind == "" {
			cur.Count, cur.Size = d.Count, d.Bytes
			continue
		}
		cur.Kinds = append(cur.Kinds, wire.AssetKindCountResponse{Kind: string(d.Kind), Count: d.Count, Size: d.Bytes})
	}
	return out
}
