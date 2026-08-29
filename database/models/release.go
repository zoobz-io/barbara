package models

import "time"

// Release is an immutable snapshot of everything live in an app. Append-only:
// never mutated, never deleted. number is monotonic per app. Rollback cuts a
// new release copying an old one's entries — the pointer never moves backward,
// so the releases table alone answers "what was live when, and who cut it".
type Release struct {
	CreatedAt time.Time `json:"created_at" db:"created_at" default:"now()"`
	ID        string    `json:"id" db:"id" constraints:"primarykey"`
	AppID     string    `json:"app_id" db:"app_id" constraints:"notnull"`
	TenantID  string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	CreatedBy string    `json:"created_by" db:"created_by" constraints:"notnull"`
	Number    int       `json:"number" db:"number" constraints:"notnull"`
}

// GetID returns the release's primary key.
func (r Release) GetID() string { return r.ID }

// Clone returns a copy of the release. It holds no reference fields, so the
// value copy is a deep copy.
func (r Release) Clone() Release { return r }

// ReleaseEntry is one live path in a release — the materialized tree, one row
// per (release, key). document_id and version_id are RESTRICT references:
// history referenced by a release survives any delete. The surrogate id exists
// for the store machinery; (release_id, key) is the real identity.
type ReleaseEntry struct {
	ID         string `json:"id" db:"id" constraints:"primarykey"`
	ReleaseID  string `json:"release_id" db:"release_id" constraints:"notnull"`
	Key        string `json:"key" db:"key" constraints:"notnull"`
	DocumentID string `json:"document_id" db:"document_id" constraints:"notnull"`
	VersionID  string `json:"version_id" db:"version_id" constraints:"notnull"`
}

// GetID returns the entry's primary key.
func (e ReleaseEntry) GetID() string { return e.ID }

// Clone returns a copy of the entry. It holds no reference fields, so the
// value copy is a deep copy.
func (e ReleaseEntry) Clone() ReleaseEntry { return e }
