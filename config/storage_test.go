package config

import "testing"

func TestStorage_Validate_Valid(t *testing.T) {
	c := Storage{Endpoint: "localhost:9000", Bucket: "barbara"}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestStorage_Validate_MissingEndpoint(t *testing.T) {
	c := Storage{Bucket: "barbara"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing endpoint")
	}
}

func TestStorage_Validate_MissingBucket(t *testing.T) {
	c := Storage{Endpoint: "localhost:9000"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing bucket")
	}
}
