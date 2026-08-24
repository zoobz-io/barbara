package config

import "testing"

func TestApp_Validate_Valid(t *testing.T) {
	c := App{APIPort: 8080, AdminPort: 8081}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestApp_Validate_MissingAPIPort(t *testing.T) {
	c := App{AdminPort: 8081}
	if err := c.Validate(); err == nil {
		t.Error("expected error for non-positive api port")
	}
}

func TestApp_Validate_MissingAdminPort(t *testing.T) {
	c := App{APIPort: 8080}
	if err := c.Validate(); err == nil {
		t.Error("expected error for non-positive admin port")
	}
}

func TestApp_Validate_SamePort(t *testing.T) {
	c := App{APIPort: 8080, AdminPort: 8080}
	if err := c.Validate(); err == nil {
		t.Error("expected error for equal api and admin ports")
	}
}
