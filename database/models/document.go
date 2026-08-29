package models

import (
	"time"

	"github.com/lib/pq"
)

// Document is the logical document: identity and system metadata, no content
// and no versioning. key is the materialized full path — derived from the
// tree (collection ancestry + name), rewritten in-transaction when an
// ancestor moves or renames — and stays the unique lookup handle.
// published_version_id points at the one published version; it drops once
// release-based publishing replaces it as the record of what is live.
//
// app_id, collection_id, and name are the tree placement. collection_id nil
// means the app root and stays nullable forever; app_id and name are pointers
// only until every write path populates them and the schema tightens to NOT
// NULL. deleted_at marks a soft-deleted document — one
// referenced by a historical release, whose key and name are freed but whose
// versions survive.
type Document struct {
	CreatedAt          time.Time      `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at" default:"now()"`
	DeletedAt          *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
	PublishedVersionID *string        `json:"published_version_id,omitempty" db:"published_version_id"`
	AppID              *string        `json:"app_id,omitempty" db:"app_id"`
	CollectionID       *string        `json:"collection_id,omitempty" db:"collection_id"`
	Name               *string        `json:"name,omitempty" db:"name"`
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
	if d.DeletedAt != nil {
		v := *d.DeletedAt
		c.DeletedAt = &v
	}
	if d.PublishedVersionID != nil {
		v := *d.PublishedVersionID
		c.PublishedVersionID = &v
	}
	if d.AppID != nil {
		v := *d.AppID
		c.AppID = &v
	}
	if d.CollectionID != nil {
		v := *d.CollectionID
		c.CollectionID = &v
	}
	if d.Name != nil {
		v := *d.Name
		c.Name = &v
	}
	if d.Tags != nil {
		c.Tags = append(pq.StringArray(nil), d.Tags...)
	}
	return c
}
