package wire

import "time"

// ReleaseResponse is the authoring API representation of a release.
type ReleaseResponse struct {
	CreatedAt time.Time `json:"created_at" description:"When the release was cut"`
	ID        string    `json:"id" description:"Release ID"`
	AppID     string    `json:"app_id" description:"Owning app"`
	TenantID  string    `json:"tenant_id" description:"Owning tenant"`
	CreatedBy string    `json:"created_by" description:"User who cut the release"`
	Number    int       `json:"number" description:"Monotonic release number within the app"`
}

// Clone returns a copy. ReleaseResponse holds no reference fields.
func (r ReleaseResponse) Clone() ReleaseResponse { return r }

// ReleaseEntryResponse is one live path in a release.
type ReleaseEntryResponse struct {
	Key        string `json:"key" description:"Live path"`
	DocumentID string `json:"document_id" description:"Document served at the path"`
	VersionID  string `json:"version_id" description:"Version served"`
}

// ReleaseWithEntriesResponse is a release together with its materialized entries.
type ReleaseWithEntriesResponse struct {
	Release ReleaseResponse        `json:"release"`
	Entries []ReleaseEntryResponse `json:"entries" description:"The release's live paths, by key"`
}

// Clone returns a deep copy.
func (r ReleaseWithEntriesResponse) Clone() ReleaseWithEntriesResponse {
	c := r
	if r.Entries != nil {
		c.Entries = make([]ReleaseEntryResponse, len(r.Entries))
		copy(c.Entries, r.Entries)
	}
	return c
}

// ReleaseListResponse is the authoring API response for listing releases.
type ReleaseListResponse struct {
	Releases []ReleaseResponse `json:"releases" description:"The app's releases, newest first"`
	Total    int               `json:"total" description:"Number of releases returned"`
	Limit    int               `json:"limit" description:"Page size"`
	Offset   int               `json:"offset" description:"Page offset"`
}

// Clone returns a deep copy.
func (r ReleaseListResponse) Clone() ReleaseListResponse {
	c := r
	if r.Releases != nil {
		c.Releases = make([]ReleaseResponse, len(r.Releases))
		copy(c.Releases, r.Releases)
	}
	return c
}
