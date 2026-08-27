package models

import "time"

// Version is an immutable, atomic snapshot of a document's full markdown. No
// diffs, never mutated: every save lands a new version with a monotonic
// version_number per document, so concurrent editors cannot destroy each
// other's work. Content is stored inline. created_by is the acting user.
type Version struct {
	CreatedAt     time.Time `json:"created_at" db:"created_at" default:"now()"`
	ID            string    `json:"id" db:"id" constraints:"primarykey"`
	DocumentID    string    `json:"document_id" db:"document_id" constraints:"notnull"`
	TenantID      string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	Content       string    `json:"content" db:"content" constraints:"notnull"`
	CreatedBy     string    `json:"created_by" db:"created_by" constraints:"notnull"`
	VersionNumber int       `json:"version_number" db:"version_number" constraints:"notnull"`
}

// GetID returns the version's primary key.
func (v Version) GetID() string { return v.ID }

// Clone returns a copy of the version. It holds no reference fields, so the
// value copy is a deep copy.
func (v Version) Clone() Version { return v }
