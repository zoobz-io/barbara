package handlers

import "github.com/zoobz-io/rocco"

// All returns every public-API handler for registration in cmd/api: the
// site-facing published reads plus the tenant-scoped authoring surface.
func All() []rocco.Endpoint {
	return []rocco.Endpoint{
		// Published reads (OpenSearch, under /published/*).
		GetPublishedDocument,
		EnumerateDocuments,
		SearchDocuments,
		// Apps.
		CreateApp,
		GetApp,
		ListApps,
		RenameApp,
		DeleteApp,
		// Collections.
		CreateCollection,
		GetCollection,
		ListAppRootContents,
		ListCollectionContents,
		RenameCollection,
		MoveCollection,
		DeleteCollection,
		// Documents authoring.
		CreateDocument,
		GetDocument,
		GetDocumentContent,
		ListDocuments,
		MoveDocument,
		DeleteDocument,
		// Tags.
		AddDocumentTag,
		RemoveDocumentTag,
		// Versions.
		SaveVersion,
		ListVersions,
		GetVersion,
		// Publishing lifecycle.
		PublishDocument,
		UnpublishDocument,
		RollbackDocument,
		// Assets.
		UploadAsset,
		GetAsset,
		ListAssets,
		DeleteAsset,
	}
}
