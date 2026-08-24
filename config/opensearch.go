package config

import "errors"

// OpenSearch holds OpenSearch connection configuration. OpenSearch is the
// serving store — site-facing pages and search read it exclusively.
type OpenSearch struct {
	Addr     string `env:"APP_OPENSEARCH_ADDR" default:"http://localhost:9200"`
	Username string `env:"APP_OPENSEARCH_USERNAME"`
	Password string `env:"APP_OPENSEARCH_PASSWORD" secret:"app/opensearch-password"`
}

// Validate checks that the configuration is valid.
func (c OpenSearch) Validate() error {
	if c.Addr == "" {
		return errors.New("opensearch address is required")
	}
	return nil
}
