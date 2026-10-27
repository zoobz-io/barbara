package handlers

import (
	"github.com/zoobz-io/openapi"
	"github.com/zoobz-io/rocco"

	"github.com/zoobz-io/barbara/internal/hast"
	"github.com/zoobz-io/barbara/internal/mdast"
)

// ConfigureOpenAPI applies the public API's OpenAPI metadata to the engine: the
// spec Info, per-tag descriptions, and the domain tag groups. Call it on the
// engine before serving so /openapi and /docs reflect it.
func ConfigureOpenAPI(e *rocco.Engine) {
	e.WithOpenAPIInfo(openapi.Info{
		Title:       "Barbara API",
		Description: "Public API for the Barbara platform: tenant-scoped authoring over apps, collections, documents, versions, releases, and assets, plus the site-facing published read surface. All endpoints require an authenticated session.",
		Version:     "0.1.0",
	})

	// Site domain — the read-only published surface, served from the search index.
	e.WithTag("Published", "Read published documents and assets exactly as released.")

	// Authoring domain — the tenant-scoped write/lifecycle surface.
	e.WithTag("Apps", "The top-level containers a tenant publishes from.")
	e.WithTag("Collections", "Folder hierarchy organizing documents within an app.")
	e.WithTag("Documents", "Authored documents, their placement, and their tags.")
	e.WithTag("Versions", "Immutable saved snapshots of a document's content.")
	e.WithTag("Publishing", "Per-document publish, unpublish, and rollback lifecycle.")
	e.WithTag("Releases", "App-wide cut points and rollback targets.")
	e.WithTag("Assets", "Uploaded binary assets and their published counterparts.")

	e.WithTagGroup("Site", "Published")
	e.WithTagGroup("Authoring", "Apps", "Collections", "Documents", "Versions", "Publishing", "Releases", "Assets")

	registerMdastModels(e)
	registerHastModels(e)
}

// registerMdastModels registers every mdast node type as an OpenAPI component.
// The published document body (format=mdast) is an mdast.MdastRoot whose children are
// a union reached only through an interface, so rocco cannot discover the
// variant types by walking the response struct — the discriminate tags refer to
// them by name, and these registrations put those named schemas in the spec.
func registerMdastModels(e *rocco.Engine) {
	e.WithModels(
		rocco.NewModel[mdast.MdastRoot](),
		rocco.NewModel[mdast.MdastParagraph](),
		rocco.NewModel[mdast.MdastHeading](),
		rocco.NewModel[mdast.MdastThematicBreak](),
		rocco.NewModel[mdast.MdastBlockquote](),
		rocco.NewModel[mdast.MdastList](),
		rocco.NewModel[mdast.MdastListItem](),
		rocco.NewModel[mdast.MdastCode](),
		rocco.NewModel[mdast.MdastHTML](),
		rocco.NewModel[mdast.MdastText](),
		rocco.NewModel[mdast.MdastEmphasis](),
		rocco.NewModel[mdast.MdastStrong](),
		rocco.NewModel[mdast.MdastDelete](),
		rocco.NewModel[mdast.MdastInlineCode](),
		rocco.NewModel[mdast.MdastBreak](),
		rocco.NewModel[mdast.MdastLink](),
		rocco.NewModel[mdast.MdastImage](),
		rocco.NewModel[mdast.MdastLinkReference](),
		rocco.NewModel[mdast.MdastImageReference](),
		rocco.NewModel[mdast.MdastDefinition](),
		rocco.NewModel[mdast.MdastFootnoteReference](),
		rocco.NewModel[mdast.MdastFootnoteDefinition](),
		rocco.NewModel[mdast.MdastTable](),
		rocco.NewModel[mdast.MdastTableRow](),
		rocco.NewModel[mdast.MdastTableCell](),
	)
}

// registerHastModels registers every hast node type as an OpenAPI component,
// for the same reason as the mdast models: the published document body
// (format=hast) is a hast.HastRoot whose children are a union reached only
// through an interface. The Hast-prefixed names keep these schemas distinct from
// the mdast ones (both trees have a root and a text node).
func registerHastModels(e *rocco.Engine) {
	e.WithModels(
		rocco.NewModel[hast.HastRoot](),
		rocco.NewModel[hast.HastElement](),
		rocco.NewModel[hast.HastText](),
		rocco.NewModel[hast.HastComment](),
		rocco.NewModel[hast.HastRaw](),
	)
}
