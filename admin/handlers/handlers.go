package handlers

import "github.com/zoobz-io/rocco"

// All returns every admin handler for registration in cmd/admin.
func All() []rocco.Endpoint {
	return []rocco.Endpoint{
		CreateDocument,
		GetDocument,
		ListDocuments,
		RenameDocument,
		DeleteDocument,
		SaveVersion,
		ListVersions,
		GetVersion,
		PublishDocument,
		UnpublishDocument,
		RollbackDocument,
	}
}
