// Package wire defines the request and response types at the public-API
// boundary. Published reads expose the document body and public metadata and
// omit internal fields (tenant_id — implicit from auth; version_id — the
// internal postgres key); authoring exposes full data, audit fields included.
package wire

import (
	"time"

	"github.com/zoobz-io/barbara/internal/mdast"
)

// PublishedDocumentResponse is the site-facing representation of a published
// document. Tenant and internal version identifiers are excluded; the document
// ID, key, tags and public version number are all a consumer needs.
//
// The document body is returned in one of two shapes, chosen by the lookup's
// format parameter. In markdown (the default) Content holds the raw markdown
// and Body and Meta are absent. In mdast Body holds the parsed tree, Meta holds
// the frontmatter, and Content is absent.
type PublishedDocumentResponse struct {
	CreatedAt     time.Time      `json:"created_at" description:"When the document was created"`
	UpdatedAt     time.Time      `json:"updated_at" description:"When the document was last updated"`
	Body          *mdast.Root    `json:"body,omitempty" description:"The document body as an mdast tree (format=mdast)"`
	Meta          map[string]any `json:"meta,omitempty" description:"Frontmatter fields (format=mdast)"`
	DocumentID    string         `json:"document_id" description:"Document ID" example:"b1e1..."`
	Key           string         `json:"key" description:"Document key" example:"guides/install.md"`
	Content       string         `json:"content,omitempty" description:"Published content as raw markdown (format=markdown)"`
	Tags          []string       `json:"tags" description:"Organizational tags"`
	VersionNumber int            `json:"version_number" description:"Published version number" example:"3"`
}

// Clone returns a copy. The mdast body is treated as immutable once built and is
// shared by reference; the tags and frontmatter maps are copied one level.
func (r PublishedDocumentResponse) Clone() PublishedDocumentResponse {
	c := r
	if r.Tags != nil {
		c.Tags = make([]string, len(r.Tags))
		copy(c.Tags, r.Tags)
	}
	if r.Meta != nil {
		c.Meta = make(map[string]any, len(r.Meta))
		for k, v := range r.Meta {
			c.Meta[k] = v
		}
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
