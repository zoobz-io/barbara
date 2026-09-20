package models

import "time"

// Release kinds: the operation that cut the release. A full-tree cut, the
// single-page publish and unpublish sugar, or a rollback copying an older
// release forward. Recorded on the row so the history reads without replaying
// events.
const (
	ReleaseKindCut       = "cut"
	ReleaseKindPublish   = "publish"
	ReleaseKindUnpublish = "unpublish"
	ReleaseKindRollback  = "rollback"
)

// Change kinds: how a path differs from the previous release. Added and removed
// are one-sided; changed is the same path at a new version; moved is the same
// document at a new path (at the same or a new version).
const (
	ChangeAdded   = "added"
	ChangeChanged = "changed"
	ChangeRemoved = "removed"
	ChangeMoved   = "moved"
)

// Release is an immutable snapshot of everything live in an app. Append-only:
// never mutated, never deleted. number is monotonic per app. Rollback cuts a
// new release copying an old one's entries — the pointer never moves backward,
// so the releases table alone answers "what was live when, and who cut it".
//
// The row also carries what the cut was and what it changed: kind, the source
// release of a rollback, the subject document of a publish or unpublish, an
// optional label, and the counts against the previous release. All of it is
// written in the cut transaction; nothing on the row is edited afterwards.
type Release struct {
	CreatedAt         time.Time `json:"created_at" db:"created_at" default:"now()"`
	Label             *string   `json:"label,omitempty" db:"label"`
	SourceReleaseID   *string   `json:"source_release_id,omitempty" db:"source_release_id"`
	SubjectDocumentID *string   `json:"subject_document_id,omitempty" db:"subject_document_id"`
	ID                string    `json:"id" db:"id" constraints:"primarykey"`
	AppID             string    `json:"app_id" db:"app_id" constraints:"notnull"`
	TenantID          string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	CreatedBy         string    `json:"created_by" db:"created_by" constraints:"notnull"`
	Kind              string    `json:"kind" db:"kind" constraints:"notnull"`
	Number            int       `json:"number" db:"number" constraints:"notnull"`
	EntryCount        int       `json:"entry_count" db:"entry_count" constraints:"notnull"`
	Added             int       `json:"added" db:"added" constraints:"notnull"`
	Changed           int       `json:"changed" db:"changed" constraints:"notnull"`
	Removed           int       `json:"removed" db:"removed" constraints:"notnull"`
	Moved             int       `json:"moved" db:"moved" constraints:"notnull"`
}

// GetID returns the release's primary key.
func (r Release) GetID() string { return r.ID }

// Clone returns a deep copy of the release.
func (r Release) Clone() Release {
	c := r
	c.Label = cloneString(r.Label)
	c.SourceReleaseID = cloneString(r.SourceReleaseID)
	c.SubjectDocumentID = cloneString(r.SubjectDocumentID)
	return c
}

// ReleaseEntry is one live path in a release — the materialized tree, one row
// per (release, key). document_id and version_id are RESTRICT references:
// history referenced by a release survives any delete. version_number is the
// served version's number, denormalized so a manifest reads without the
// versions table. The surrogate id exists for the store machinery;
// (release_id, key) is the real identity.
type ReleaseEntry struct {
	ID            string `json:"id" db:"id" constraints:"primarykey"`
	ReleaseID     string `json:"release_id" db:"release_id" constraints:"notnull"`
	Key           string `json:"key" db:"key" constraints:"notnull"`
	DocumentID    string `json:"document_id" db:"document_id" constraints:"notnull"`
	VersionID     string `json:"version_id" db:"version_id" constraints:"notnull"`
	VersionNumber int    `json:"version_number" db:"version_number" constraints:"notnull"`
}

// GetID returns the entry's primary key.
func (e ReleaseEntry) GetID() string { return e.ID }

// Clone returns a copy of the entry. It holds no reference fields, so the
// value copy is a deep copy.
func (e ReleaseEntry) Clone() ReleaseEntry { return e }

// ReleaseChange is one path that differs between a release and the one before
// it — the materialized diff, one row per (release, key). Key and document are
// the path as of this release (for a removal, the path that went away). The
// previous side is nil for an addition; the new side is nil for a removal; a
// move carries the previous key. Entries remain the manifest; these rows are a
// projection of two adjacent manifests, recomputable from them.
type ReleaseChange struct {
	PrevKey           *string `json:"prev_key,omitempty" db:"prev_key"`
	PrevVersionID     *string `json:"prev_version_id,omitempty" db:"prev_version_id"`
	PrevVersionNumber *int    `json:"prev_version_number,omitempty" db:"prev_version_number"`
	VersionID         *string `json:"version_id,omitempty" db:"version_id"`
	VersionNumber     *int    `json:"version_number,omitempty" db:"version_number"`
	ID                string  `json:"id" db:"id" constraints:"primarykey"`
	ReleaseID         string  `json:"release_id" db:"release_id" constraints:"notnull"`
	Key               string  `json:"key" db:"key" constraints:"notnull"`
	DocumentID        string  `json:"document_id" db:"document_id" constraints:"notnull"`
	Change            string  `json:"change" db:"change" constraints:"notnull"`
}

// GetID returns the change's primary key.
func (c ReleaseChange) GetID() string { return c.ID }

// Clone returns a deep copy of the change.
func (c ReleaseChange) Clone() ReleaseChange {
	d := c
	d.PrevKey = cloneString(c.PrevKey)
	d.PrevVersionID = cloneString(c.PrevVersionID)
	d.PrevVersionNumber = cloneInt(c.PrevVersionNumber)
	d.VersionID = cloneString(c.VersionID)
	d.VersionNumber = cloneInt(c.VersionNumber)
	return d
}

// cloneString copies an optional string.
func cloneString(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

// cloneInt copies an optional int.
func cloneInt(n *int) *int {
	if n == nil {
		return nil
	}
	v := *n
	return &v
}
