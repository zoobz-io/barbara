package wire

// AssetResponse is the authoring API metadata for a stored asset. The bytes are
// served separately by the download endpoint; this carries metadata only.
type AssetResponse struct {
	Key         string `json:"key" description:"Asset key, unique per app" example:"images/logo.png"`
	ContentType string `json:"content_type" description:"Stored MIME type" example:"image/png"`
	Size        int64  `json:"size" description:"Size in bytes"`
}

// Clone returns a deep copy.
func (r AssetResponse) Clone() AssetResponse { return r }

// AssetListResponse is the authoring API response for listing a tenant's assets.
type AssetListResponse struct {
	Assets []AssetResponse `json:"assets" description:"The app's assets"`
	Total  int             `json:"total" description:"Number of assets returned"`
}

// Clone returns a deep copy.
func (r AssetListResponse) Clone() AssetListResponse {
	c := r
	if r.Assets != nil {
		c.Assets = make([]AssetResponse, len(r.Assets))
		copy(c.Assets, r.Assets)
	}
	return c
}
