package config

import "errors"

var (
	errMeshAddressRequired = errors.New("mesh address is required")
	errMeshCACertRequired  = errors.New("CA cert path is required when TLS is enabled")
	errMeshCertKeyPair     = errors.New("cert_file and key_file must both be set or both empty")
)

// Mesh holds the connection to janus/aegis over the service mesh. Barbara has
// no user table — it delegates identity and entitlement to janus, and services
// authenticate to each other with mesh CA client certificates.
//
// Address is where barbara dials janus/aegis. The TLS fields mirror the house
// mesh-credentials contract: when Enabled is false (default), connections use
// insecure credentials for local dev; with only CACert, server-only TLS; with
// CACert + CertFile + KeyFile, mutual TLS.
type Mesh struct {
	Address  string `env:"APP_MESH_ADDR" default:"localhost:9443"`
	CACert   string `env:"APP_GRPC_TLS_CA_CERT"`
	CertFile string `env:"APP_GRPC_TLS_CERT_FILE"`
	KeyFile  string `env:"APP_GRPC_TLS_KEY_FILE"`
	Enabled  bool   `env:"APP_GRPC_TLS_ENABLED" default:"false"`
}

// Validate checks that the configuration is consistent.
func (c Mesh) Validate() error {
	if c.Address == "" {
		return errMeshAddressRequired
	}
	if !c.Enabled {
		return nil
	}
	if c.CACert == "" {
		return errMeshCACertRequired
	}
	// CertFile and KeyFile must both be set or both empty.
	if (c.CertFile == "") != (c.KeyFile == "") {
		return errMeshCertKeyPair
	}
	return nil
}

// MutualTLS returns true if both client cert and key are configured.
func (c Mesh) MutualTLS() bool {
	return c.Enabled && c.CertFile != "" && c.KeyFile != ""
}
