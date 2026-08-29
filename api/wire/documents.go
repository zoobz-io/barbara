package wire

import (
	"errors"
	"time"
)

// CreateDocumentRequest is the body for creating a document in the tree.
// collection_id is null (or omitted) to place at the app root.
type CreateDocumentRequest struct {
	CollectionID *string `json:"collection_id" description:"Parent collection ID, or null for the app root"`
	Name         string  `json:"name" description:"Document name, unique among siblings" example:"install.md"`
}

// Validate requires a non-empty name. Value receiver so rocco's value-typed
// Validatable check picks it up.
func (r CreateDocumentRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// MoveDocumentRequest is the body for moving a document: a new parent collection
// (null = app root) and a new name.
type MoveDocumentRequest struct {
	CollectionID *string `json:"collection_id" description:"New parent collection ID, or null for the app root"`
	Name         string  `json:"name" description:"New name, unique among siblings" example:"setup.md"`
}

// Validate requires a non-empty name.
func (r MoveDocumentRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// AddTagRequest is the body for adding a tag to a document.
type AddTagRequest struct {
	Tag string `json:"tag" description:"Organizational tag to add" example:"guide"`
}

// Clone returns a deep copy.
func (r AddTagRequest) Clone() AddTagRequest { return r }

// DocumentResponse is the authoring API representation of a document.
type DocumentResponse struct {
	CreatedAt          time.Time `json:"created_at" description:"Creation timestamp"`
	UpdatedAt          time.Time `json:"updated_at" description:"Last update timestamp"`
	PublishedVersionID *string   `json:"published_version_id,omitempty" description:"The published version, if any"`
	ID                 string    `json:"id" description:"Document ID" example:"b1e1..."`
	TenantID           string    `json:"tenant_id" description:"Owning tenant"`
	Key                string    `json:"key" description:"Document key" example:"guides/install.md"`
	Status             string    `json:"status" description:"Lifecycle status: draft or published" example:"published"`
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

// DocumentContentResponse is a document together with its head (latest) version's
// content — the single-request read behind opening a document for editing.
// Content is null when the document has no versions yet (an empty document).
type DocumentContentResponse struct {
	Content  *ContentBlock    `json:"content" description:"The head version's content, or null if the document has no versions"`
	Document DocumentResponse `json:"document"`
}

// ContentBlock is the head version carried on a DocumentContentResponse.
type ContentBlock struct {
	CreatedAt     time.Time `json:"created_at" description:"When the head version was saved"`
	VersionID     string    `json:"version_id" description:"Head version ID"`
	Content       string    `json:"content" description:"Head version content"`
	VersionNumber int       `json:"version_number" description:"Head version number"`
}

// Clone returns a deep copy.
func (r DocumentContentResponse) Clone() DocumentContentResponse {
	c := r
	c.Document = r.Document.Clone()
	if r.Content != nil {
		b := *r.Content
		c.Content = &b
	}
	return c
}

// DocumentListResponse is the authoring API response for listing documents.
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
