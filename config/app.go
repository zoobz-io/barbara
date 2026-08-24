// Package config provides typed configuration structs loaded from environment variables.
package config

import "errors"

// App holds application-level configuration. Barbara runs one binary per
// surface, so App carries both ports: the public API (site-facing) and the
// admin API (authoring).
type App struct {
	APIPort   int `env:"APP_PORT" default:"8080"`
	AdminPort int `env:"APP_ADMIN_PORT" default:"8081"`
}

// Validate checks that the configuration is valid.
func (c App) Validate() error {
	if c.APIPort <= 0 {
		return errors.New("api port must be positive")
	}
	if c.AdminPort <= 0 {
		return errors.New("admin port must be positive")
	}
	if c.APIPort == c.AdminPort {
		return errors.New("api and admin ports must differ")
	}
	return nil
}
