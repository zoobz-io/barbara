package models

import (
	"strings"
	"time"
)

// Asset is a binary blob held in object storage, addressed by a key that is
// unique per tenant. Assets have no Postgres row and no versioning — putting the
// same key overwrites the bytes. Data carries the blob on a get; it is empty in
// a listing, which returns metadata only.
type Asset struct {
	// LastModified is when the object was last written, as the bucket reports
	// it. Zero when the source did not carry one.
	LastModified time.Time
	Key          string
	ContentType  string
	Data         []byte
	Size         int64
}

// AssetFolder is a subfolder seen from one level of an app's asset tree: the
// segment name, how many assets sit anywhere beneath it, and their total
// size. Folders are a convention over key prefixes — nothing is stored for
// them.
type AssetFolder struct {
	// LastWrittenAt is when an asset beneath the folder was last written, as
	// the bookkeeping recorded it; nil when the rows have none.
	LastWrittenAt *time.Time
	Name          string
	Count         int
	Size          int64
}

// AssetLevel is one level of an app's asset tree — the direct subfolders and
// the assets whose key sits exactly at Path (metadata only, no bytes). Path is
// "" for the root, otherwise the folder without a trailing slash.
type AssetLevel struct {
	Path    string
	Folders []AssetFolder
	Assets  []*Asset
}

// Kind is an asset's media family — the one vocabulary the API, the stats,
// and the studio's icons and previews share. It is derived from the content
// type by KindOf and never stored on the object.
type Kind string

// The kinds, mirroring the studio's file-icon families one for one.
const (
	KindImage       Kind = "image"
	KindVideo       Kind = "video"
	KindAudio       Kind = "audio"
	KindCode        Kind = "code"
	KindSpreadsheet Kind = "spreadsheet"
	KindArchive     Kind = "archive"
	KindText        Kind = "text"
	KindFile        Kind = "file" // the fallback: anything not otherwise placed
)

// Kinds lists every kind in display order.
var Kinds = []Kind{KindImage, KindVideo, KindAudio, KindCode, KindSpreadsheet, KindArchive, KindText, KindFile}

// Code-like text and application types that read as source files.
var codeTypes = map[string]struct{}{
	"text/css":               {},
	"text/html":              {},
	"text/javascript":        {},
	"application/javascript": {},
	"application/json":       {},
	"application/xml":        {},
	"text/xml":               {},
	"application/yaml":       {},
	"text/yaml":              {},
}

// Spreadsheet and delimited-data types.
var spreadsheetTypes = map[string]struct{}{
	"text/csv":                  {},
	"text/tab-separated-values": {},
	"application/vnd.ms-excel":  {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": {},
}

// Archive and compressed types.
var archiveTypes = map[string]struct{}{
	"application/zip":              {},
	"application/gzip":             {},
	"application/x-tar":            {},
	"application/x-7z-compressed":  {},
	"application/x-rar-compressed": {},
}

// KindOf classifies a content type. Families (image, video, audio) match on
// the type's prefix; code, spreadsheet and archive on known subtypes; PDF and
// any remaining text/* are text; everything else is the plain file. Any
// parameters (charset) are ignored. The order matters: text/css is code, not
// text, because the subtype check runs before the text/* fallback.
func KindOf(contentType string) Kind {
	mediaType, _, _ := strings.Cut(contentType, ";")
	t := strings.ToLower(strings.TrimSpace(mediaType))
	switch {
	case strings.HasPrefix(t, "image/"):
		return KindImage
	case strings.HasPrefix(t, "video/"):
		return KindVideo
	case strings.HasPrefix(t, "audio/"):
		return KindAudio
	}
	if _, ok := codeTypes[t]; ok {
		return KindCode
	}
	if _, ok := spreadsheetTypes[t]; ok {
		return KindSpreadsheet
	}
	if _, ok := archiveTypes[t]; ok {
		return KindArchive
	}
	if t == "application/pdf" || strings.HasPrefix(t, "text/") {
		return KindText
	}
	return KindFile
}
