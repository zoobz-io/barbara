package wire

import "time"

// SaveVersionRequest is the body for saving a new version.
type SaveVersionRequest struct {
	Content string `json:"content" description:"The document's full markdown content"`
}

// VersionResponse is the admin API representation of a version.
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

// VersionListResponse is the admin API response for listing versions.
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
