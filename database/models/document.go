package models

import (
	"time"

	"github.com/lib/pq"
)

// Document is the logical document: identity and system metadata, no content
// and no versioning. key is the materialized full path — derived from the
// tree (collection ancestry + name), rewritten in-transaction when an
// ancestor moves or renames — and stays the unique lookup handle, per app.
// What is live is recorded by the app's current release, not on this row.
//
// app_id, collection_id, and name are the tree placement. collection_id nil
// means the app root. deleted_at marks a soft-deleted document — one
// referenced by a historical release, whose key and name are freed but whose
// versions survive.
type Document struct {
	CreatedAt    time.Time      `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at" default:"now()"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
	CollectionID *string        `json:"collection_id,omitempty" db:"collection_id"`
	ID           string         `json:"id" db:"id" constraints:"primarykey"`
	TenantID     string         `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	AppID        string         `json:"app_id" db:"app_id" constraints:"notnull"`
	Name         string         `json:"name" db:"name" constraints:"notnull"`
	Key          string         `json:"key" db:"key" constraints:"notnull"`
	Tags         pq.StringArray `json:"tags" db:"tags"`
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
	if d.CollectionID != nil {
		v := *d.CollectionID
		c.CollectionID = &v
	}
	if d.Tags != nil {
		c.Tags = append(pq.StringArray(nil), d.Tags...)
	}
	return c
}
