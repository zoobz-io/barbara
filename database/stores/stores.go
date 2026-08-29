package stores

import (
	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/grub"
)

// Stores is the aggregate of every data-access store, constructed once in boot
// and shared across all surfaces. Multi-store writes with atomicity invariants
// live as transactional methods here, never composed from individual store
// calls at call sites.
//
// It grows as capabilities land: Documents (#13), Versions (#14), Assets (#17)
// add their stores to this struct.
type Stores struct {
	Documents *Documents
	Versions  *Versions
	Jobs      *Jobs
	Search    *Search
	Assets    *Assets

	db       *sqlx.DB       // for transactional aggregate methods (publishing)
	renderer astql.Renderer // for aggregate methods that build ad-hoc table handles (backfill)
}

// New constructs the stores aggregate. The SQL stores share the Postgres
// connection and astql renderer; the search store wraps the OpenSearch
// provider; the assets store wraps the object-storage bucket.
func New(db *sqlx.DB, renderer astql.Renderer, search grub.SearchProvider, bucket grub.BucketProvider) *Stores {
	documents := NewDocuments(db, renderer)
	versions := NewVersions(db, renderer, documents)
	// Documents enriches reads with the head version (GetWithHead/ListWithHead),
	// so it needs the versions store. Wired here to break the construction cycle:
	// versions already depends on documents for the parent-row lock.
	documents.versions = versions
	return &Stores{
		Documents: documents,
		Versions:  versions,
		Jobs:      NewJobs(db, renderer),
		Search:    NewSearch(search),
		Assets:    NewAssets(bucket),
		db:        db,
		renderer:  renderer,
	}
}
