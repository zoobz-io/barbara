// Package transformers holds pure conversions between domain models. Unlike the
// per-surface transformers (admin/api), these operate below the API boundary —
// e.g. building the OpenSearch projection the publish path writes.
package transformers

import (
	"strings"

	"github.com/zoobz-io/barbara/database/models"
)

// Projection merges a document's system metadata with a version's content into
// the DocumentIndex — the OpenSearch projection of a published document. app_id
// and parent_path are materialized here (parent_path is the key's folder), so
// the live index answers app scoping and folder listing without a tree walk.
func Projection(doc *models.Document, version *models.Version) models.DocumentIndex {
	appID := ""
	if doc.AppID != nil {
		appID = *doc.AppID
	}
	return models.DocumentIndex{
		DocumentID:    doc.ID,
		TenantID:      doc.TenantID,
		Key:           doc.Key,
		AppID:         appID,
		ParentPath:    parentPath(doc.Key),
		Tags:          []string(doc.Tags),
		VersionID:     version.ID,
		VersionNumber: version.VersionNumber,
		Content:       version.Content,
		CreatedAt:     doc.CreatedAt,
		UpdatedAt:     doc.UpdatedAt,
	}
}

// parentPath is the folder portion of a key — everything before the last slash,
// or "" for a root key.
func parentPath(key string) string {
	if i := strings.LastIndex(key, "/"); i >= 0 {
		return key[:i]
	}
	return ""
}
