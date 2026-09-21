// Package transformers maps domain models to public-API wire types. Pure
// functions, no side effects. Published reads drop internal fields (tenant_id,
// version_id) — the wire type simply has no field for them; authoring responses
// expose full data, audit fields included.
package transformers

import (
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/zoobz-io/barbara/api/wire"
	"github.com/zoobz-io/barbara/database/models"
)

// urlScheme matches a URL that opens with a scheme (https:, mailto:, and the
// like) — a colon in the scheme position, before any path separator.
var urlScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)

// PublishedAssetResolver returns the URL resolver a published document's mdast
// tree is rewritten with (see mdast.Rewrite). It turns a relative asset path
// into the published asset route, so a site fetches a URL rather than a path
// that only means something inside the app's asset tree.
//
// A URL that is already fetchable on its own is left unchanged: one with a
// scheme, one rooted at "/", and an in-page fragment ("#..."). Any other URL is
// a path relative to the document's folder (parentPath, "" at the app root); it
// is resolved against that folder — "../" segments included — and returned as
// /published/apps/{appID}/assets/object?key=<resolved key>.
func PublishedAssetResolver(appID, parentPath string) func(string) string {
	return func(u string) string {
		if u == "" || strings.HasPrefix(u, "/") || strings.HasPrefix(u, "#") || urlScheme.MatchString(u) {
			return u
		}
		key := path.Join(parentPath, u)
		return "/published/apps/" + appID + "/assets/object?key=" + assetKeyQuery(key)
	}
}

// assetKeyQuery escapes a resolved asset key for use as a query value while
// keeping its path separators readable.
func assetKeyQuery(key string) string {
	return strings.ReplaceAll(url.QueryEscape(key), "%2F", "/")
}

// IndexToResponse maps a document projection to its site-facing response.
func IndexToResponse(d *models.DocumentIndex) wire.PublishedDocumentResponse {
	return wire.PublishedDocumentResponse{
		DocumentID:    d.DocumentID,
		Key:           d.Key,
		Content:       d.Content,
		Tags:          d.Tags,
		VersionNumber: d.VersionNumber,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// IndexesToListResponse maps a page of projections plus its total to the
// site-facing list response.
func IndexesToListResponse(docs []models.DocumentIndex, total int64, limit, offset int) wire.PublishedDocumentListResponse {
	out := wire.PublishedDocumentListResponse{
		Documents: make([]wire.PublishedDocumentResponse, len(docs)),
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}
	for i := range docs {
		out.Documents[i] = IndexToResponse(&docs[i])
	}
	return out
}
