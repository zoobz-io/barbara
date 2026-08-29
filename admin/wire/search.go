// Package wire defines the request and response types at the admin (internal
// platform) API boundary. Admin is cross-tenant, so responses expose the owning
// tenant.
package wire

import "time"

// SearchResultResponse is one cross-tenant search hit. Unlike the site-facing
// reads, it exposes tenant_id — admin is internal and sees across tenants.
type SearchResultResponse struct {
	CreatedAt     time.Time `json:"created_at" description:"When the document was created"`
	DocumentID    string    `json:"document_id" description:"Document ID"`
	TenantID      string    `json:"tenant_id" description:"Owning tenant"`
	Key           string    `json:"key" description:"Document key" example:"guides/install.md"`
	Content       string    `json:"content" description:"Published content"`
	Tags          []string  `json:"tags" description:"Organizational tags"`
	VersionNumber int       `json:"version_number" description:"Published version number"`
}

// Clone returns a deep copy.
func (r SearchResultResponse) Clone() SearchResultResponse {
	c := r
	if r.Tags != nil {
		c.Tags = make([]string, len(r.Tags))
		copy(c.Tags, r.Tags)
	}
	return c
}

// SearchResultsResponse is a page of cross-tenant search hits.
type SearchResultsResponse struct {
	Results []SearchResultResponse `json:"results" description:"The page of cross-tenant matches"`
	Total   int64                  `json:"total" description:"Total number of matches across all pages"`
	Limit   int                    `json:"limit" description:"Page size"`
	Offset  int                    `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r SearchResultsResponse) Clone() SearchResultsResponse {
	c := r
	if r.Results != nil {
		c.Results = make([]SearchResultResponse, len(r.Results))
		for i, res := range r.Results {
			c.Results[i] = res.Clone()
		}
	}
	return c
}
