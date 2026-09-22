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
	"strings"

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
func patch(spec *openapi.OpenAPI) {
	if spec.Components == nil || spec.Components.Schemas == nil {
		return
	}

	// rocco (through v0.1.23) inlined validation-error details as an array of
	// ValidationFieldError and emitted a $ref to that named schema without
	// adding it to components. Backfill it when a rocco version omits it.
	if _, ok := spec.Components.Schemas["ValidationFieldError"]; !ok {
		spec.Components.Schemas["ValidationFieldError"] = &openapi.Schema{
			Type: openapi.NewSchemaType("object"),
			Properties: map[string]*openapi.Schema{
				"field":   {Type: openapi.NewSchemaType("string"), Description: "The field that failed validation"},
				"message": {Type: openapi.NewSchemaType("string"), Description: "Description of the validation failure"},
			},
			Required: []string{"field", "message"},
		}
	}

	// A map[string]any value type (the mdast frontmatter meta) has no schema,
	// so rocco emits additionalProperties as a $ref to an undefined
	// "interface {}" schema. Turn it into a free-form object so the spec is
	// self-contained.
	for _, schema := range spec.Components.Schemas {
		openFreeFormObjects(schema)
	}

	markMdastNullable(spec.Components.Schemas)
}

// markMdastNullable marks the mdast node fields that are present-and-null as
// nullable. rocco emits a Go pointer field as its base type with no null
// allowance, but several mdast fields are always present and often null — an
// absent code language, an unchecked list item, a titleless link, an unaligned
// table column. Without this the fixtures fail schema validation and the SDK
// types omit the null. The set mirrors the pointer fields in internal/mdast,
// and the fixture-validation test guards it against drift.
func markMdastNullable(schemas map[string]*openapi.Schema) {
	nullable := map[string][]string{
		"Code":       {"lang", "meta"},
		"List":       {"start"},
		"ListItem":   {"checked"},
		"Link":       {"title"},
		"Image":      {"title"},
		"Definition": {"title"},
	}
	for name, props := range nullable {
		s := schemas[name]
		if s == nil {
			continue
		}
		for _, p := range props {
			makeNullable(s.Properties[p])
		}
	}
	// A table's align holds one entry per column, each "left"/"right"/"center"
	// or null for the default.
	if t := schemas["Table"]; t != nil {
		if align, ok := t.Properties["align"]; ok {
			makeNullable(align.Items)
		}
	}
}

// makeNullable adds "null" to a schema's type.
func makeNullable(s *openapi.Schema) {
	if s == nil || s.Type == nil {
		return
	}
	types := s.Type.Strings()
	for _, t := range types {
		if t == "null" {
			return
		}
	}
	s.Type = openapi.NewSchemaTypes(append(types, "null"))
}

// openFreeFormObjects rewrites a dangling additionalProperties $ref to
// "interface {}" — rocco's rendering of a map[string]any value — into an open
// object (additionalProperties: true).
func openFreeFormObjects(s *openapi.Schema) {
	if s == nil {
		return
	}
	if ap, ok := s.AdditionalProperties.(*openapi.Schema); ok {
		if strings.HasSuffix(ap.Ref, "/interface {}") {
			s.AdditionalProperties = true
		} else {
			openFreeFormObjects(ap)
		}
	}
	for _, p := range s.Properties {
		openFreeFormObjects(p)
	}
	openFreeFormObjects(s.Items)
	for _, o := range s.OneOf {
		openFreeFormObjects(o)
	}
	for _, o := range s.AnyOf {
		openFreeFormObjects(o)
	}
	for _, o := range s.AllOf {
		openFreeFormObjects(o)
	}
}
