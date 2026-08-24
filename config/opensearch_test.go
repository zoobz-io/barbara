package config

import "testing"

func TestOpenSearch_Validate_Valid(t *testing.T) {
	c := OpenSearch{Addr: "http://localhost:9200"}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestOpenSearch_Validate_MissingAddr(t *testing.T) {
	c := OpenSearch{}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing addr")
	}
}
