package config

import "testing"

func TestMesh_Validate_Disabled(t *testing.T) {
	c := Mesh{Address: "localhost:9443", Enabled: false}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestMesh_Validate_MissingAddress(t *testing.T) {
	c := Mesh{Enabled: false}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing address")
	}
}

func TestMesh_Validate_ServerOnly(t *testing.T) {
	c := Mesh{Address: "localhost:9443", Enabled: true, CACert: "/certs/ca.pem"}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestMesh_Validate_MutualTLS(t *testing.T) {
	c := Mesh{
		Address:  "localhost:9443",
		Enabled:  true,
		CACert:   "/certs/ca.pem",
		CertFile: "/certs/client.pem",
		KeyFile:  "/certs/client-key.pem",
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestMesh_Validate_MissingCACert(t *testing.T) {
	c := Mesh{Address: "localhost:9443", Enabled: true}
	if err := c.Validate(); err == nil {
		t.Error("expected error for missing CA cert")
	}
}

func TestMesh_Validate_CertWithoutKey(t *testing.T) {
	c := Mesh{Address: "localhost:9443", Enabled: true, CACert: "/certs/ca.pem", CertFile: "/certs/client.pem"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for cert without key")
	}
}

func TestMesh_Validate_KeyWithoutCert(t *testing.T) {
	c := Mesh{Address: "localhost:9443", Enabled: true, CACert: "/certs/ca.pem", KeyFile: "/certs/client-key.pem"}
	if err := c.Validate(); err == nil {
		t.Error("expected error for key without cert")
	}
}

func TestMesh_MutualTLS(t *testing.T) {
	tests := []struct {
		name string
		cfg  Mesh
		want bool
	}{
		{"disabled", Mesh{Address: "localhost:9443", Enabled: false}, false},
		{"server only", Mesh{Address: "localhost:9443", Enabled: true, CACert: "/ca.pem"}, false},
		{"mtls", Mesh{Address: "localhost:9443", Enabled: true, CACert: "/ca.pem", CertFile: "/c.pem", KeyFile: "/k.pem"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.MutualTLS(); got != tt.want {
				t.Errorf("MutualTLS() = %v, want %v", got, tt.want)
			}
		})
	}
}
