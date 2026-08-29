package wire

import (
	"errors"
	"time"
)

// CreateCollectionRequest is the body for creating a collection. parent_id is
// null (or omitted) to create at the app root.
type CreateCollectionRequest struct {
	ParentID *string `json:"parent_id" description:"Parent collection ID, or null for the app root"`
	Name     string  `json:"name" description:"Collection name, unique among siblings" example:"guides"`
}

// Validate requires a non-empty name. Value receiver so rocco's value-typed
// Validatable check picks it up.
func (r CreateCollectionRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// RenameCollectionRequest is the body for renaming a collection.
type RenameCollectionRequest struct {
	Name string `json:"name" description:"New name, unique among siblings" example:"tutorials"`
}

// Validate requires a non-empty name.
func (r RenameCollectionRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// MoveCollectionRequest is the body for moving a collection. parent_id is null
// (or omitted) to move to the app root — a nil target is valid, so there is
// nothing to validate.
type MoveCollectionRequest struct {
	ParentID *string `json:"parent_id" description:"New parent collection ID, or null for the app root"`
}

// CollectionResponse is the authoring API representation of a collection.
type CollectionResponse struct {
	CreatedAt time.Time `json:"created_at" description:"Creation timestamp"`
	UpdatedAt time.Time `json:"updated_at" description:"Last update timestamp"`
	ParentID  *string   `json:"parent_id,omitempty" description:"Parent collection ID, or null at the app root"`
	ID        string    `json:"id" description:"Collection ID"`
	TenantID  string    `json:"tenant_id" description:"Owning tenant"`
	AppID     string    `json:"app_id" description:"Owning app"`
	Name      string    `json:"name" description:"Collection name" example:"guides"`
}

// Clone returns a deep copy.
func (r CollectionResponse) Clone() CollectionResponse {
	c := r
	if r.ParentID != nil {
		v := *r.ParentID
		c.ParentID = &v
	}
	return c
}

// CollectionContentsResponse is a collection's direct children — subcollections
// and documents together, each document carrying its derived status.
type CollectionContentsResponse struct {
	Subcollections []CollectionResponse `json:"subcollections" description:"Direct subcollections, by name"`
	Documents      []DocumentResponse   `json:"documents" description:"Direct documents, by key"`
}

// Clone returns a deep copy.
func (r CollectionContentsResponse) Clone() CollectionContentsResponse {
	c := r
	if r.Subcollections != nil {
		c.Subcollections = make([]CollectionResponse, len(r.Subcollections))
		for i, s := range r.Subcollections {
			c.Subcollections[i] = s.Clone()
		}
	}
	if r.Documents != nil {
		c.Documents = make([]DocumentResponse, len(r.Documents))
		for i, d := range r.Documents {
			c.Documents[i] = d.Clone()
		}
	}
	return c
}
