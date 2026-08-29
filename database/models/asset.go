package models

// Asset is a binary blob held in object storage, addressed by a key that is
// unique per tenant. Assets have no Postgres row and no versioning — putting the
// same key overwrites the bytes. Data carries the blob on a get; it is empty in
// a listing, which returns metadata only.
type Asset struct {
	Key         string
	ContentType string
	Data        []byte
	Size        int64
}
