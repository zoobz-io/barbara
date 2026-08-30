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
//
// key is the SERVING path — the release entry's key, not the document's. The
// two diverge when a document moves or renames after a release is cut: the
// authoring key changes, but the release (and therefore the live site) keeps
// the path it recorded until the next cut. Passing doc.Key here is only
// correct when the caller has just derived the entry from the live tree.
func Projection(doc *models.Document, version *models.Version, key string) models.DocumentIndex {
	return models.DocumentIndex{
		DocumentID:    doc.ID,
		TenantID:      doc.TenantID,
		AppID:         doc.AppID,
		Key:           key,
		ParentPath:    parentPath(key),
		Tags:          []string(doc.Tags),
		VersionID:     version.ID,
		VersionNumber: version.VersionNumber,
		Content:       version.Content,
		CreatedAt:     doc.CreatedAt,
		UpdatedAt:     doc.UpdatedAt,
	}
}

// parentPath is the folder portion of a key — everything before the last slash,
// or "" for a root key. The empty-string root convention is load-bearing: the
// folder read resolves an absent path parameter to "" and must match what is
// written here.
func parentPath(key string) string {
	if i := strings.LastIndex(key, "/"); i >= 0 {
		return key[:i]
	}
	return ""
}
