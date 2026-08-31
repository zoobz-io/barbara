package handlers

import (
	"github.com/zoobz-io/openapi"
	"github.com/zoobz-io/rocco"
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
}
