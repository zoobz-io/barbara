// Package wire defines the request and response types at the site-facing (api)
// boundary. The read surface serves published content to mesh services: it
// exposes the document body and its public metadata, and omits internal fields
// (tenant_id — implicit from auth; version_id — the internal postgres key).
package wire

import "time"

// PublishedDocumentResponse is the site-facing representation of a published
// document. Tenant and internal version identifiers are excluded; the document
// ID, key, content, tags and public version number are all a consumer needs.
type PublishedDocumentResponse struct {
	CreatedAt     time.Time `json:"created_at" description:"When the document was created"`
	UpdatedAt     time.Time `json:"updated_at" description:"When the document was last updated"`
	DocumentID    string    `json:"document_id" description:"Document ID" example:"b1e1..."`
	Key           string    `json:"key" description:"Document key" example:"guides/install.md"`
	Content       string    `json:"content" description:"Published content"`
	Tags          []string  `json:"tags" description:"Organizational tags"`
	VersionNumber int       `json:"version_number" description:"Published version number" example:"3"`
}

// Clone returns a deep copy.
func (r PublishedDocumentResponse) Clone() PublishedDocumentResponse {
	c := r
	if r.Tags != nil {
		c.Tags = make([]string, len(r.Tags))
		copy(c.Tags, r.Tags)
	}
	return c
}

// PublishedDocumentListResponse is the site-facing response for enumerate and
// search — a page of published documents plus the total match count.
type PublishedDocumentListResponse struct {
	Documents []PublishedDocumentResponse `json:"documents" description:"The page of published documents"`
	Total     int64                       `json:"total" description:"Total number of matches across all pages"`
	Limit     int                         `json:"limit" description:"Page size"`
	Offset    int                         `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r PublishedDocumentListResponse) Clone() PublishedDocumentListResponse {
	c := r
	if r.Documents != nil {
		c.Documents = make([]PublishedDocumentResponse, len(r.Documents))
		for i, d := range r.Documents {
			c.Documents[i] = d.Clone()
		}
	}
	return c
}
