package wire

import (
	"testing"
	"time"
)

// Clone deep-copies the stats: the timestamps, the kind and day slices, and
// each day's kinds are independent of the original's.
func TestAssetStatsResponse_Clone(t *testing.T) {
	at := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	orig := AssetStatsResponse{
		ComputedAt:    &at,
		LastWrittenAt: &at,
		Kinds:         []AssetKindStatsResponse{{Kind: "image", Count: 1, LastWrittenAt: &at}},
		Days:          []AssetDayStatsResponse{{Day: "2026-09-16", Kinds: []AssetKindCountResponse{{Kind: "image", Count: 1}}}},
	}
	cp := orig.Clone()
	if cp.ComputedAt == orig.ComputedAt || cp.LastWrittenAt == orig.LastWrittenAt || cp.Kinds[0].LastWrittenAt == orig.Kinds[0].LastWrittenAt {
		t.Fatal("Clone shares a timestamp pointer")
	}
	*cp.ComputedAt = at.Add(time.Hour)
	cp.Kinds[0].Kind = "video"
	cp.Days[0].Kinds[0].Kind = "video"
	if !orig.ComputedAt.Equal(at) || orig.Kinds[0].Kind != "image" || orig.Days[0].Kinds[0].Kind != "image" {
		t.Errorf("mutating the copy changed the original: %+v", orig)
	}

	// Empty stats clone to empty (nil stays nil).
	empty := AssetStatsResponse{}.Clone()
	if empty.ComputedAt != nil || empty.Kinds != nil || empty.Days != nil {
		t.Errorf("empty stats clone got non-nil fields: %+v", empty)
	}
}

// Clone deep-copies a folder level: the subfolder and asset slices are
// independent of the original's.
func TestAssetFolderResponse_Clone(t *testing.T) {
	orig := AssetFolderResponse{
		Path:    "images",
		Folders: []AssetSubfolderResponse{{Name: "icons", Count: 2}},
		Assets:  []AssetResponse{{Key: "images/logo.png"}},
	}
	cp := orig.Clone()
	cp.Folders[0].Name = "changed"
	cp.Assets[0].Key = "changed"
	if orig.Folders[0].Name != "icons" || orig.Assets[0].Key != "images/logo.png" {
		t.Errorf("Clone did not isolate the slices: %+v", orig)
	}
}

// Clone deep-copies an asset and a subfolder: their timestamps are
// independent of the original's, and a list clone isolates each element.
func TestAssetResponse_Clone(t *testing.T) {
	at := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	asset := AssetResponse{Key: "a.png", LastModified: &at}
	if cp := asset.Clone(); cp.LastModified == asset.LastModified {
		t.Error("asset Clone shares the timestamp pointer")
	}
	folder := AssetSubfolderResponse{Name: "images", LastWrittenAt: &at}
	if cp := folder.Clone(); cp.LastWrittenAt == folder.LastWrittenAt {
		t.Error("subfolder Clone shares the timestamp pointer")
	}
	list := AssetListResponse{Assets: []AssetResponse{asset}}
	cp := list.Clone()
	*cp.Assets[0].LastModified = at.Add(time.Hour)
	if !list.Assets[0].LastModified.Equal(at) {
		t.Error("list Clone shares an element's timestamp")
	}
	if (AssetResponse{}).Clone().LastModified != nil {
		t.Error("an asset without a timestamp cloned to one")
	}
}

// A move needs a key; the request and a kind count are plain values, so
// their clones are copies.
func TestMoveAssetRequest_Validate(t *testing.T) {
	if err := (MoveAssetRequest{}).Validate(); err == nil {
		t.Error("an empty key validated")
	}
	if err := (MoveAssetRequest{Key: "images/logo.png"}).Validate(); err != nil {
		t.Errorf("a valid key was rejected: %v", err)
	}
	if cp := (MoveAssetRequest{Key: "a"}).Clone(); cp.Key != "a" {
		t.Errorf("request Clone = %+v", cp)
	}
	orig := AssetKindCountResponse{Kind: "image", Count: 2, Size: 3}
	if cp := orig.Clone(); cp != orig {
		t.Errorf("kind count Clone = %+v, want %+v", cp, orig)
	}
}
