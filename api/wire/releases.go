package wire

import (
	"errors"
	"time"
	"unicode/utf8"
)

// MaxReleaseLabelLen bounds a release label: a short note, not release notes.
const MaxReleaseLabelLen = 200

// CutReleaseRequest is the optional body for cutting or rolling back to a
// release: a label fixed at cut time. An empty body is a release with no label.
type CutReleaseRequest struct {
	Label string `json:"label,omitempty" description:"Optional short label for the release, fixed at cut time" example:"Launch of the 2.3 docs"`
}

// Validate bounds the label. Value receiver so rocco's value-typed Validatable
// check picks it up.
func (r CutReleaseRequest) Validate() error {
	if utf8.RuneCountInString(r.Label) > MaxReleaseLabelLen {
		return errors.New("label must be at most 200 characters")
	}
	return nil
}

// ReleaseResponse is the authoring API representation of a release: its
// identity, what kind of cut it was, and the counts against the previous
// release, so a list of releases reads at a glance without loading entries.
type ReleaseResponse struct {
	CreatedAt         time.Time `json:"created_at" description:"When the release was cut"`
	Label             *string   `json:"label,omitempty" description:"The label given at cut time, if any"`
	SourceReleaseID   *string   `json:"source_release_id,omitempty" description:"For a rollback, the release whose entries were copied forward"`
	SubjectDocumentID *string   `json:"subject_document_id,omitempty" description:"For a publish or unpublish, the document the release was about"`
	ID                string    `json:"id" description:"Release ID"`
	AppID             string    `json:"app_id" description:"Owning app"`
	TenantID          string    `json:"tenant_id" description:"Owning tenant"`
	CreatedBy         string    `json:"created_by" description:"User who cut the release"`
	Kind              string    `json:"kind" description:"What cut the release: cut (full tree), publish, unpublish, or rollback" example:"cut"`
	Number            int       `json:"number" description:"Monotonic release number within the app"`
	EntryCount        int       `json:"entry_count" description:"Live paths in the release"`
	Added             int       `json:"added" description:"Paths added since the previous release"`
	Changed           int       `json:"changed" description:"Paths at a new version since the previous release"`
	Removed           int       `json:"removed" description:"Paths removed since the previous release"`
	Moved             int       `json:"moved" description:"Documents at a new path since the previous release"`
}

// Clone returns a deep copy.
func (r ReleaseResponse) Clone() ReleaseResponse {
	c := r
	c.Label = cloneString(r.Label)
	c.SourceReleaseID = cloneString(r.SourceReleaseID)
	c.SubjectDocumentID = cloneString(r.SubjectDocumentID)
	return c
}

// ReleaseEntryResponse is one live path in a release.
type ReleaseEntryResponse struct {
	Key           string `json:"key" description:"Live path"`
	DocumentID    string `json:"document_id" description:"Document served at the path"`
	VersionID     string `json:"version_id" description:"Version served"`
	VersionNumber int    `json:"version_number" description:"The served version's number within its document"`
}

// ReleaseWithEntriesResponse is a release together with its materialized entries.
type ReleaseWithEntriesResponse struct {
	Entries []ReleaseEntryResponse `json:"entries" description:"The release's live paths, by key"`
	Release ReleaseResponse        `json:"release"`
}

// Clone returns a deep copy.
func (r ReleaseWithEntriesResponse) Clone() ReleaseWithEntriesResponse {
	c := r
	c.Release = r.Release.Clone()
	if r.Entries != nil {
		c.Entries = make([]ReleaseEntryResponse, len(r.Entries))
		copy(c.Entries, r.Entries)
	}
	return c
}

// ReleaseChangeResponse is one document that differs between a release and the
// one before it. The previous side is absent for an addition, the new side for
// a removal; a move carries the previous key.
type ReleaseChangeResponse struct {
	PrevKey           *string `json:"prev_key,omitempty" description:"For a move, the path before this release"`
	PrevVersionID     *string `json:"prev_version_id,omitempty" description:"The version served before this release, absent for an addition"`
	PrevVersionNumber *int    `json:"prev_version_number,omitempty" description:"Number of the version served before, absent for an addition"`
	VersionID         *string `json:"version_id,omitempty" description:"The version this release serves, absent for a removal"`
	VersionNumber     *int    `json:"version_number,omitempty" description:"Number of the version this release serves, absent for a removal"`
	Key               string  `json:"key" description:"The path as of this release (for a removal, the path that went away)"`
	DocumentID        string  `json:"document_id" description:"The document"`
	Change            string  `json:"change" description:"added, changed, removed, or moved" example:"changed"`
}

// Clone returns a deep copy.
func (r ReleaseChangeResponse) Clone() ReleaseChangeResponse {
	c := r
	c.PrevKey = cloneString(r.PrevKey)
	c.PrevVersionID = cloneString(r.PrevVersionID)
	c.PrevVersionNumber = cloneInt(r.PrevVersionNumber)
	c.VersionID = cloneString(r.VersionID)
	c.VersionNumber = cloneInt(r.VersionNumber)
	return c
}

// ReleaseChangesResponse is a release together with its changes against the
// previous release.
type ReleaseChangesResponse struct {
	Changes []ReleaseChangeResponse `json:"changes" description:"What differs from the previous release, by key"`
	Release ReleaseResponse         `json:"release"`
}

// Clone returns a deep copy.
func (r ReleaseChangesResponse) Clone() ReleaseChangesResponse {
	c := r
	c.Release = r.Release.Clone()
	if r.Changes != nil {
		c.Changes = make([]ReleaseChangeResponse, len(r.Changes))
		for i, ch := range r.Changes {
			c.Changes[i] = ch.Clone()
		}
	}
	return c
}

// ReleaseListResponse is the authoring API response for listing releases.
type ReleaseListResponse struct {
	Releases []ReleaseResponse `json:"releases" description:"The app's releases, newest first"`
	Total    int               `json:"total" description:"Total releases in the app, across all pages"`
	Limit    int               `json:"limit" description:"Page size"`
	Offset   int               `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r ReleaseListResponse) Clone() ReleaseListResponse {
	c := r
	if r.Releases != nil {
		c.Releases = make([]ReleaseResponse, len(r.Releases))
		for i, rel := range r.Releases {
			c.Releases[i] = rel.Clone()
		}
	}
	return c
}

// cloneString copies an optional string.
func cloneString(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

// cloneInt copies an optional int.
func cloneInt(n *int) *int {
	if n == nil {
		return nil
	}
	v := *n
	return &v
}
