// Package transformers holds pure conversions between domain models. Unlike the
// per-surface transformers (admin/api), these operate below the API boundary —
// e.g. building the OpenSearch projection the publish path writes.
package transformers

import (
	"github.com/zoobz-io/barbara/database/models"
)

// Projection merges a document's system metadata with a version's content into
// the DocumentIndex — the OpenSearch projection of a published document.
func Projection(doc *models.Document, version *models.Version) models.DocumentIndex {
	return models.DocumentIndex{
		DocumentID:    doc.ID,
		TenantID:      doc.TenantID,
		Key:           doc.Key,
		Tags:          []string(doc.Tags),
		VersionID:     version.ID,
		VersionNumber: version.VersionNumber,
		Content:       version.Content,
		CreatedAt:     doc.CreatedAt,
		UpdatedAt:     doc.UpdatedAt,
	}
}
