package models

import "testing"

// KindOf mirrors the studio's icon families: prefix families first, then the
// subtype sets, then PDF and text, then the plain file. Parameters and case
// are ignored.
func TestKindOf(t *testing.T) {
	cases := map[string]Kind{
		"image/png":                 KindImage,
		"image/svg+xml":             KindImage,
		"video/mp4":                 KindVideo,
		"audio/mpeg":                KindAudio,
		"text/css":                  KindCode, // subtype set wins over text/*
		"application/json":          KindCode,
		"application/yaml":          KindCode,
		"text/csv":                  KindSpreadsheet,
		"application/vnd.ms-excel":  KindSpreadsheet,
		"application/zip":           KindArchive,
		"application/gzip":          KindArchive,
		"application/pdf":           KindText,
		"text/markdown":             KindText,
		"text/plain; charset=utf-8": KindText,
		"Image/JPEG":                KindImage,
		"application/octet-stream":  KindFile,
		"font/woff2":                KindFile,
		"":                          KindFile,
	}
	for ct, want := range cases {
		if got := KindOf(ct); got != want {
			t.Errorf("KindOf(%q) = %q, want %q", ct, got, want)
		}
	}
}

func TestKinds_Complete(t *testing.T) {
	seen := map[Kind]bool{}
	for _, k := range Kinds {
		if seen[k] {
			t.Errorf("Kinds lists %q twice", k)
		}
		seen[k] = true
	}
	if len(Kinds) != 8 {
		t.Errorf("Kinds has %d entries, want 8", len(Kinds))
	}
}
