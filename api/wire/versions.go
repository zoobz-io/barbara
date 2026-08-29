package wire

import (
	"errors"
	"time"
)

// SaveVersionRequest is the body for saving a new version. base_version is the
// version number the client edited from (0 for the first version); the save is
// rejected with 409 if it is no longer the document's head.
type SaveVersionRequest struct {
	BaseVersion *int   `json:"base_version" description:"The head version the edit was based on (0 for the first version)"`
	Content     string `json:"content" description:"The document's full markdown content"`
}

// Validate requires base_version — a pointer so 0 (the first-version case) is
// distinguishable from an omitted field. Value receiver so rocco's value-typed
// Validatable check picks it up.
func (r SaveVersionRequest) Validate() error {
	if r.BaseVersion == nil {
		return errors.New("base_version is required")
	}
	if *r.BaseVersion < 0 {
		return errors.New("base_version must be >= 0")
	}
	return nil
}

// VersionResponse is the authoring API representation of a version.
type VersionResponse struct {
	CreatedAt     time.Time `json:"created_at" description:"Creation timestamp"`
	ID            string    `json:"id" description:"Version ID"`
	DocumentID    string    `json:"document_id" description:"Parent document ID"`
	TenantID      string    `json:"tenant_id" description:"Owning tenant"`
	Content       string    `json:"content" description:"The version's markdown content"`
	CreatedBy     string    `json:"created_by" description:"User who saved this version"`
	VersionNumber int       `json:"version_number" description:"Monotonic number within the document"`
}

// Clone returns a copy. VersionResponse holds no reference fields, so the value
// copy is a deep copy.
func (r VersionResponse) Clone() VersionResponse { return r }

// VersionListResponse is the authoring API response for listing versions.
type VersionListResponse struct {
	Versions []VersionResponse `json:"versions" description:"The document's versions, newest first"`
	Total    int               `json:"total" description:"Number of versions returned"`
	Limit    int               `json:"limit" description:"Page size"`
	Offset   int               `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r VersionListResponse) Clone() VersionListResponse {
	c := r
	if r.Versions != nil {
		c.Versions = make([]VersionResponse, len(r.Versions))
		copy(c.Versions, r.Versions)
	}
	return c
}
