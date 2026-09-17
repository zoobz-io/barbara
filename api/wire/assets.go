package wire

import (
	"errors"
	"time"
)

// AssetResponse is the authoring API metadata for a stored asset. The bytes are
// served separately by the download endpoint; this carries metadata only.
type AssetResponse struct {
	LastModified *time.Time `json:"last_modified,omitempty" description:"When the object was last written, as object storage reports it; omitted when unknown"`
	Key          string     `json:"key" description:"Asset key, unique per app" example:"images/logo.png"`
	ContentType  string     `json:"content_type" description:"Stored MIME type" example:"image/png"`
	Kind         string     `json:"kind" description:"Media family derived from the content type: image, video, audio, code, spreadsheet, archive, text, or file" example:"image"`
	Size         int64      `json:"size" description:"Size in bytes"`
}

// Clone returns a deep copy.
func (r AssetResponse) Clone() AssetResponse {
	c := r
	if r.LastModified != nil {
		t := *r.LastModified
		c.LastModified = &t
	}
	return c
}

// AssetListResponse is the authoring API response for listing a tenant's assets.
type AssetListResponse struct {
	Assets []AssetResponse `json:"assets" description:"The app's assets"`
	Total  int             `json:"total" description:"Number of assets returned"`
}

// Clone returns a deep copy.
func (r AssetListResponse) Clone() AssetListResponse {
	c := r
	if r.Assets != nil {
		c.Assets = make([]AssetResponse, len(r.Assets))
		for i, a := range r.Assets {
			c.Assets[i] = a.Clone()
		}
	}
	return c
}

// AssetSubfolderResponse is a subfolder at one level of an app's asset tree.
type AssetSubfolderResponse struct {
	LastWrittenAt *time.Time `json:"last_written_at,omitempty" description:"When an asset beneath the folder was last written; omitted when unknown"`
	Name          string     `json:"name" description:"Folder name (one key segment)" example:"images"`
	Count         int        `json:"count" description:"Assets anywhere beneath the folder"`
	Size          int64      `json:"size" description:"Total bytes of the assets beneath the folder"`
}

// Clone returns a deep copy.
func (r AssetSubfolderResponse) Clone() AssetSubfolderResponse {
	c := r
	if r.LastWrittenAt != nil {
		t := *r.LastWrittenAt
		c.LastWrittenAt = &t
	}
	return c
}

// AssetFolderResponse is one level of an app's asset tree: the folder's direct
// subfolders and the assets directly inside it. Folders are key prefixes by
// convention, so this is the browsing view over a flat key space.
type AssetFolderResponse struct {
	Path    string                   `json:"path" description:"The folder, empty for the root" example:"images"`
	Folders []AssetSubfolderResponse `json:"folders" description:"Direct subfolders, by name"`
	Assets  []AssetResponse          `json:"assets" description:"Assets directly in the folder, by key"`
}

// Clone returns a deep copy.
func (r AssetFolderResponse) Clone() AssetFolderResponse {
	c := r
	if r.Folders != nil {
		c.Folders = make([]AssetSubfolderResponse, len(r.Folders))
		for i, f := range r.Folders {
			c.Folders[i] = f.Clone()
		}
	}
	if r.Assets != nil {
		c.Assets = make([]AssetResponse, len(r.Assets))
		for i, a := range r.Assets {
			c.Assets[i] = a.Clone()
		}
	}
	return c
}

// MoveAssetRequest names the key an asset moves to: another folder, another
// name, or both.
type MoveAssetRequest struct {
	Key string `json:"key" description:"The new key, segments joined by slashes" example:"images/brand/logo.png"`
}

// Validate requires a key.
func (r MoveAssetRequest) Validate() error {
	if r.Key == "" {
		return errors.New("key is required")
	}
	return nil
}

// Clone returns a copy.
func (r MoveAssetRequest) Clone() MoveAssetRequest { return r }

// CreateAssetFolderRequest names the folder to create.
type CreateAssetFolderRequest struct {
	Path string `json:"path" description:"The folder path, segments joined by slashes" example:"images/icons"`
}

// AssetStatsResponse is the app-level view of its assets: the totals, the
// breakdown by media family, and writes per UTC day. Every number is kept by
// bookkeeping rows that each write and delete adjusts, never a live count of
// the bucket; computed_at is when they were last rebuilt from it, absent until
// the first rebuild.
type AssetStatsResponse struct {
	ComputedAt    *time.Time               `json:"computed_at,omitempty" description:"When the bookkeeping was last rebuilt from object storage; omitted until the first rebuild"`
	LastWrittenAt *time.Time               `json:"last_written_at,omitempty" description:"When an asset was last written; omitted for an app without assets"`
	Kinds         []AssetKindStatsResponse `json:"kinds" description:"The breakdown by media family, by kind; only kinds with assets appear"`
	Days          []AssetDayStatsResponse  `json:"days" description:"Assets by the UTC day they were last written, oldest first, up to 90 days back; days without a write are absent"`
	Count         int64                    `json:"count" description:"Total assets"`
	Size          int64                    `json:"size" description:"Total bytes"`
}

// Clone returns a deep copy.
func (r AssetStatsResponse) Clone() AssetStatsResponse {
	c := r
	if r.ComputedAt != nil {
		t := *r.ComputedAt
		c.ComputedAt = &t
	}
	if r.LastWrittenAt != nil {
		t := *r.LastWrittenAt
		c.LastWrittenAt = &t
	}
	if r.Kinds != nil {
		c.Kinds = make([]AssetKindStatsResponse, len(r.Kinds))
		for i, k := range r.Kinds {
			c.Kinds[i] = k.Clone()
		}
	}
	if r.Days != nil {
		c.Days = make([]AssetDayStatsResponse, len(r.Days))
		for i, d := range r.Days {
			c.Days[i] = d.Clone()
		}
	}
	return c
}

// AssetKindStatsResponse is the rollup of one media family.
type AssetKindStatsResponse struct {
	LastWrittenAt *time.Time `json:"last_written_at,omitempty" description:"When an asset of the kind was last written"`
	Kind          string     `json:"kind" description:"Media family: image, video, audio, code, spreadsheet, archive, text, or file" example:"image"`
	Count         int64      `json:"count" description:"Assets of the kind"`
	Size          int64      `json:"size" description:"Bytes of the kind"`
}

// Clone returns a deep copy.
func (r AssetKindStatsResponse) Clone() AssetKindStatsResponse {
	c := r
	if r.LastWrittenAt != nil {
		t := *r.LastWrittenAt
		c.LastWrittenAt = &t
	}
	return c
}

// AssetDayStatsResponse is one UTC day of the series: the assets whose last
// write fell on it, in total and by kind. An overwrite moves an asset to the
// day it was rewritten; a delete removes it from its day.
type AssetDayStatsResponse struct {
	Day   string                   `json:"day" description:"The UTC calendar day" example:"2026-09-16"`
	Kinds []AssetKindCountResponse `json:"kinds" description:"The day's assets by kind"`
	Count int64                    `json:"count" description:"Assets last written that day"`
	Size  int64                    `json:"size" description:"Their bytes"`
}

// Clone returns a deep copy.
func (r AssetDayStatsResponse) Clone() AssetDayStatsResponse {
	c := r
	if r.Kinds != nil {
		c.Kinds = make([]AssetKindCountResponse, len(r.Kinds))
		copy(c.Kinds, r.Kinds)
	}
	return c
}

// AssetKindCountResponse is one kind's share of a day.
type AssetKindCountResponse struct {
	Kind  string `json:"kind" description:"Media family" example:"image"`
	Count int64  `json:"count" description:"Assets of the kind"`
	Size  int64  `json:"size" description:"Bytes of the kind"`
}

// Clone returns a copy.
func (r AssetKindCountResponse) Clone() AssetKindCountResponse { return r }
