package config

import "errors"

// Storage holds MinIO/S3 object storage configuration. Assets — binary blobs
// referenced by markdown — live in the grub bucket, not Postgres or OpenSearch.
type Storage struct {
	Endpoint  string `env:"APP_STORAGE_ENDPOINT" default:"localhost:9000"`
	AccessKey string `env:"APP_STORAGE_ACCESS_KEY" secret:"app/storage-access-key"`
	SecretKey string `env:"APP_STORAGE_SECRET_KEY" secret:"app/storage-secret-key"`
	Bucket    string `env:"APP_STORAGE_BUCKET" default:"barbara"`
	UseSSL    bool   `env:"APP_STORAGE_USE_SSL" default:"false"`
}

// Validate checks that the configuration is valid.
func (c Storage) Validate() error {
	if c.Endpoint == "" {
		return errors.New("storage endpoint is required")
	}
	if c.Bucket == "" {
		return errors.New("storage bucket is required")
	}
	return nil
}
