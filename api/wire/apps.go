package wire

import (
	"errors"
	"time"
)

// CreateAppRequest is the body for creating an app.
type CreateAppRequest struct {
	Name string `json:"name" description:"App name, unique per tenant" example:"docs-site"`
}

// Validate requires a non-empty name. Value receiver so rocco's value-typed
// Validatable check picks it up.
func (r CreateAppRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// RenameAppRequest is the body for renaming an app.
type RenameAppRequest struct {
	Name string `json:"name" description:"New name, unique per tenant" example:"marketing-site"`
}

// Validate requires a non-empty name.
func (r RenameAppRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// AppResponse is the authoring API representation of an app.
type AppResponse struct {
	CreatedAt        time.Time `json:"created_at" description:"Creation timestamp"`
	UpdatedAt        time.Time `json:"updated_at" description:"Last update timestamp"`
	CurrentReleaseID *string   `json:"current_release_id,omitempty" description:"The live release, if any"`
	ID               string    `json:"id" description:"App ID"`
	TenantID         string    `json:"tenant_id" description:"Owning tenant"`
	Name             string    `json:"name" description:"App name" example:"docs-site"`
}

// Clone returns a deep copy.
func (r AppResponse) Clone() AppResponse {
	c := r
	if r.CurrentReleaseID != nil {
		v := *r.CurrentReleaseID
		c.CurrentReleaseID = &v
	}
	return c
}

// AppListResponse is the authoring API response for listing apps.
type AppListResponse struct {
	Apps   []AppResponse `json:"apps" description:"The tenant's apps"`
	Total  int           `json:"total" description:"Number of apps returned"`
	Limit  int           `json:"limit" description:"Page size"`
	Offset int           `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r AppListResponse) Clone() AppListResponse {
	c := r
	if r.Apps != nil {
		c.Apps = make([]AppResponse, len(r.Apps))
		for i, a := range r.Apps {
			c.Apps[i] = a.Clone()
		}
	}
	return c
}
