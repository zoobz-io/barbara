package models

import "time"

// DocumentIndex is the OpenSearch document type — the serving-store projection
// of a published document. It merges the postgres document's system metadata
// (key, tags) with the published version's content, and is indexed with the
// document ID as the OpenSearch doc ID: one live entry per published document,
// replaced on update, removed on unpublish or delete.
//
// The site-facing surface reads this projection exclusively, which is what
// makes full-text search over exclusively-published content free — the search
// index is the serving store, no filtering, no second lookup.
type DocumentIndex struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Key       string    `json:"key"`
	Content   string    `json:"content"`
	TenantID  string    `json:"tenant_id"`
	AppID     string    `json:"app_id"`
	// ParentPath is the key's folder — the key minus its last segment — and ""
	// for a document at the app root. Materialized at projection time so a
	// folder listing is one term query, never a tree walk. The empty-string
	// root convention is load-bearing: the folder read resolves an absent path
	// parameter to "" and must match what the projection writes.
	ParentPath    string   `json:"parent_path"`
	DocumentID    string   `json:"document_id"`
	VersionID     string   `json:"version_id"`
	Tags          []string `json:"tags"`
	VersionNumber int      `json:"version_number"`
}

// Clone returns a deep copy of the index document.
func (d DocumentIndex) Clone() DocumentIndex {
	c := d
	if d.Tags != nil {
		c.Tags = make([]string, len(d.Tags))
		copy(c.Tags, d.Tags)
	}
	return c
}
