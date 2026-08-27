package models

import (
	"time"

	"github.com/lib/pq"
)

// Document is the logical document: identity and system metadata, no content
// and no versioning. key is user-supplied and unique per tenant, opaque even
// when path-like; every query is tenant-scoped. published_version_id points at
// the one published version, and is nil until the document is first published.
type Document struct {
	CreatedAt          time.Time      `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at" default:"now()"`
	PublishedVersionID *string        `json:"published_version_id,omitempty" db:"published_version_id"`
	ID                 string         `json:"id" db:"id" constraints:"primarykey"`
	TenantID           string         `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	Key                string         `json:"key" db:"key" constraints:"notnull"`
	Tags               pq.StringArray `json:"tags" db:"tags"`
}

// GetID returns the document's primary key.
func (d Document) GetID() string { return d.ID }

// Clone returns a deep copy of the document.
func (d Document) Clone() Document {
	c := d
	if d.PublishedVersionID != nil {
		v := *d.PublishedVersionID
		c.PublishedVersionID = &v
	}
	if d.Tags != nil {
		c.Tags = append(pq.StringArray(nil), d.Tags...)
	}
	return c
}
