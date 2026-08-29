package models

import "time"

// Collection is a folder: identity, name, parent — nothing else. It never
// learns about its contents; no child counts, no rollups. parent_id nil means
// the app root. app_id is denormalized onto every row so scoping never walks
// to the root. Sibling names are unique per (app, parent), a namespace shared
// with sibling documents — the cross-table half of that check lives in the
// store transaction, not the schema.
type Collection struct {
	CreatedAt time.Time `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" default:"now()"`
	ParentID  *string   `json:"parent_id,omitempty" db:"parent_id"`
	ID        string    `json:"id" db:"id" constraints:"primarykey"`
	TenantID  string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	AppID     string    `json:"app_id" db:"app_id" constraints:"notnull"`
	Name      string    `json:"name" db:"name" constraints:"notnull"`
}

// GetID returns the collection's primary key.
func (c Collection) GetID() string { return c.ID }

// Clone returns a deep copy of the collection.
func (c Collection) Clone() Collection {
	cp := c
	if c.ParentID != nil {
		v := *c.ParentID
		cp.ParentID = &v
	}
	return cp
}
