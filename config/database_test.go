package config

import "testing"

func validDatabase() Database {
	return Database{
		Host:     "localhost",
		Name:     "barbara",
		User:     "barbara",
		Password: "secret",
		SSLMode:  "disable",
		Port:     5432,
	}
}

func TestDatabase_Validate_Valid(t *testing.T) {
	if err := validDatabase().Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestDatabase_Validate_MissingHost(t *testing.T) {
	c := validDatabase()
	c.Host = ""
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing host")
	}
}

func TestDatabase_Validate_NonPositivePort(t *testing.T) {
	c := validDatabase()
	c.Port = 0
	if err := c.Validate(); err == nil {
		t.Error("expected error for non-positive port")
	}
}

func TestDatabase_Validate_MissingName(t *testing.T) {
	c := validDatabase()
	c.Name = ""
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestDatabase_Validate_MissingUser(t *testing.T) {
	c := validDatabase()
	c.User = ""
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing user")
	}
}

func TestDatabase_DSN(t *testing.T) {
	c := validDatabase()
	want := "host=localhost port=5432 user=barbara password=secret dbname=barbara sslmode=disable"
	if got := c.DSN(); got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}
