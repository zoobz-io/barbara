package handlers

import "github.com/zoobz-io/rocco"

// All returns every site-facing handler for registration in cmd/api.
func All() []rocco.Endpoint {
	return []rocco.Endpoint{
		GetPublishedDocument,
		EnumerateDocuments,
		SearchDocuments,
	}
}
