package models

import (
	"testing"
	"time"
)

// The bookkeeping rows expose their primary key, and the ones with an
// optional timestamp clone without sharing it.
func TestAssetBookkeepingRows_Clone(t *testing.T) {
	at := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)

	folder := AssetFolderStat{ID: "f-1", Path: "images", LastWrittenAt: &at}
	if folder.GetID() != "f-1" {
		t.Errorf("folder GetID = %q", folder.GetID())
	}
	fc := folder.Clone()
	if fc.LastWrittenAt == folder.LastWrittenAt {
		t.Error("folder Clone shares the timestamp pointer")
	}
	*fc.LastWrittenAt = at.Add(time.Hour)
	if !folder.LastWrittenAt.Equal(at) {
		t.Error("mutating the folder copy changed the original")
	}
	if (AssetFolderStat{}).Clone().LastWrittenAt != nil {
		t.Error("a folder without a timestamp cloned to one")
	}

	kind := AssetKindStat{ID: "k-1", Kind: KindImage, LastWrittenAt: &at}
	if kind.GetID() != "k-1" {
		t.Errorf("kind GetID = %q", kind.GetID())
	}
	kc := kind.Clone()
	if kc.LastWrittenAt == kind.LastWrittenAt {
		t.Error("kind Clone shares the timestamp pointer")
	}
	*kc.LastWrittenAt = at.Add(time.Hour)
	if !kind.LastWrittenAt.Equal(at) {
		t.Error("mutating the kind copy changed the original")
	}
	if (AssetKindStat{}).Clone().LastWrittenAt != nil {
		t.Error("a kind without a timestamp cloned to one")
	}

	day := AssetDayStat{ID: "d-1", Day: at, Kind: KindImage, Count: 2, Bytes: 3}
	if day.GetID() != "d-1" || day.Clone() != day {
		t.Errorf("day row: GetID = %q, Clone = %+v", day.GetID(), day.Clone())
	}

	mark := AssetBookkeeping{ID: "b-1", ComputedAt: at}
	if mark.GetID() != "b-1" || mark.Clone() != mark {
		t.Errorf("bookkeeping row: GetID = %q, Clone = %+v", mark.GetID(), mark.Clone())
	}
}
