package handlers

import (
	"github.com/zoobz-io/openapi"
	"github.com/zoobz-io/rocco"
)

// ConfigureOpenAPI applies the admin API's OpenAPI metadata to the engine: the
// spec Info and per-tag descriptions. Call it on the engine before serving so
// /openapi and /docs reflect it.
func ConfigureOpenAPI(e *rocco.Engine) {
	e.WithOpenAPIInfo(openapi.Info{
		Title:       "Barbara Admin API",
		Description: "Internal platform API for Barbara operators: tenant-agnostic capabilities gated behind an admin entitlement. Cross-tenant search is the surface seeded here.",
		Version:     "0.1.0",
	})

	e.WithTag("Admin", "Cross-tenant platform operations.")
}
