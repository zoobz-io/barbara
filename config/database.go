package config

import (
	"errors"
	"fmt"
)

// Database holds PostgreSQL connection configuration. Postgres is barbara's
// system of record — documents, versions, tags, and the jobs table.
type Database struct {
	Host     string `env:"APP_DB_HOST" default:"localhost"`
	Name     string `env:"APP_DB_NAME" default:"barbara"`
	User     string `env:"APP_DB_USER" default:"barbara"`
	Password string `env:"APP_DB_PASSWORD" secret:"app/db-password"`
	SSLMode  string `env:"APP_DB_SSLMODE" default:"disable"`
	Port     int    `env:"APP_DB_PORT" default:"5432"`
}

// DSN returns a PostgreSQL connection string.
func (c Database) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// Validate checks that the configuration is valid.
func (c Database) Validate() error {
	if c.Host == "" {
		return errors.New("database host is required")
	}
	if c.Port <= 0 {
		return errors.New("database port must be positive")
	}
	if c.Name == "" {
		return errors.New("database name is required")
	}
	if c.User == "" {
		return errors.New("database user is required")
	}
	return nil
}
