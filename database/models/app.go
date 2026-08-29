package models

import "time"

// App is the release unit. A tenant owns many apps; an app owns a collection
// tree and a linear release history. current_release_id points at the live
// release and is the only mutable pointer in the system — its history is the
// releases table. No metadata beyond name yet; columns get added when real
// requirements name them.
type App struct {
	CreatedAt        time.Time `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at" default:"now()"`
	CurrentReleaseID *string   `json:"current_release_id,omitempty" db:"current_release_id"`
	ID               string    `json:"id" db:"id" constraints:"primarykey"`
	TenantID         string    `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	Name             string    `json:"name" db:"name" constraints:"notnull"`
}

// GetID returns the app's primary key.
func (a App) GetID() string { return a.ID }

// Clone returns a deep copy of the app.
func (a App) Clone() App {
	c := a
	if a.CurrentReleaseID != nil {
		v := *a.CurrentReleaseID
		c.CurrentReleaseID = &v
	}
	return c
}
