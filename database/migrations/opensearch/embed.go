// Package opensearch embeds the OpenSearch index mapping files. These are the
// serving-store schema — a separate mechanism from the goose SQL migrations,
// which only handle Postgres. Boot applies them via EnsureIndices.
package opensearch

import "embed"

// Mappings contains all JSON mapping files for OpenSearch indices, named
// NNN_index.json — the numeric prefix orders application, the rest is the
// index name.
//
//go:embed *.json
var Mappings embed.FS
