// Package openapispec dumps a surface's OpenAPI specification to disk without
// starting a server or touching the database. It builds a bare rocco engine,
// applies the same OpenAPI metadata and endpoint set the surface's server uses,
// and marshals the generated spec. The output feeds the SDK client generators
// in the web monorepo (web/packages/*-sdk).
//
// Barbara has two surfaces, so the dump-and-patch logic lives here once and
// cmd/apispec and cmd/adminspec stay thin.
package openapispec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zoobz-io/openapi"
	"github.com/zoobz-io/rocco"
)

// Dump generates a surface's OpenAPI spec and writes it as indented JSON to
// out, creating the directory as needed.
//
// A bare engine carries no DB, DI, or auth — endpoint registration only records
// handler metadata (paths, request/response types, error defs), and each
// handler resolves its dependencies lazily at request time. That is all
// GenerateOpenAPI needs, so no runtime boot is required.
func Dump(configure func(*rocco.Engine), endpoints []rocco.Endpoint, out string) error {
	e := rocco.NewEngine()
	configure(e)
	e.WithHandlers(endpoints...)

	spec := e.GenerateOpenAPI(nil)
	patch(spec)

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}

// patch repairs known gaps in rocco's generated spec so the output is
// self-contained and validates against strict OpenAPI tooling.
//
// rocco (through v0.1.23) inlines validation-error details as an array of
// ValidationFieldError and emits a $ref to that named schema, but never adds
// the schema to components. Backfill it here from the known rocco type shape.
func patch(spec *openapi.OpenAPI) {
	if spec.Components == nil || spec.Components.Schemas == nil {
		return
	}
	if _, ok := spec.Components.Schemas["ValidationFieldError"]; ok {
		return
	}
	spec.Components.Schemas["ValidationFieldError"] = &openapi.Schema{
		Type: openapi.NewSchemaType("object"),
		Properties: map[string]*openapi.Schema{
			"field":   {Type: openapi.NewSchemaType("string"), Description: "The field that failed validation"},
			"message": {Type: openapi.NewSchemaType("string"), Description: "Description of the validation failure"},
		},
		Required: []string{"field", "message"},
	}
}
