package wire

// PublishRequest is the body for publishing (or rolling back to) a version.
type PublishRequest struct {
	VersionID string `json:"version_id" description:"The version to publish"`
}
