// Package wire defines the request and response types at the admin API
// boundary. Admin exposes full data — no masking — including audit fields.
package wire

import "time"

// CreateDocumentRequest is the body for creating a document.
type CreateDocumentRequest struct {
	Key string `json:"key" description:"User-supplied key, unique per tenant" example:"guides/install.md"`
}

// RenameDocumentRequest is the body for renaming a document.
type RenameDocumentRequest struct {
	Key string `json:"key" description:"New key, unique per tenant" example:"guides/setup.md"`
}

// DocumentResponse is the admin API representation of a document.
type DocumentResponse struct {
	CreatedAt          time.Time `json:"created_at" description:"Creation timestamp"`
	UpdatedAt          time.Time `json:"updated_at" description:"Last update timestamp"`
	PublishedVersionID *string   `json:"published_version_id,omitempty" description:"The published version, if any"`
	ID                 string    `json:"id" description:"Document ID" example:"b1e1..."`
	TenantID           string    `json:"tenant_id" description:"Owning tenant"`
	Key                string    `json:"key" description:"Document key" example:"guides/install.md"`
	Tags               []string  `json:"tags" description:"Organizational tags"`
}

// Clone returns a deep copy.
func (r DocumentResponse) Clone() DocumentResponse {
	c := r
	if r.PublishedVersionID != nil {
		v := *r.PublishedVersionID
		c.PublishedVersionID = &v
	}
	if r.Tags != nil {
		c.Tags = make([]string, len(r.Tags))
		copy(c.Tags, r.Tags)
	}
	return c
}

// DocumentListResponse is the admin API response for listing documents.
type DocumentListResponse struct {
	Documents []DocumentResponse `json:"documents" description:"The tenant's documents"`
	Total     int                `json:"total" description:"Number of documents returned"`
	Limit     int                `json:"limit" description:"Page size"`
	Offset    int                `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r DocumentListResponse) Clone() DocumentListResponse {
	c := r
	if r.Documents != nil {
		c.Documents = make([]DocumentResponse, len(r.Documents))
		for i, d := range r.Documents {
			c.Documents[i] = d.Clone()
		}
	}
	return c
}
