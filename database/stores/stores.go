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

	db *sqlx.DB // for transactional aggregate methods (publishing)
}

// New constructs the stores aggregate. The SQL stores share the Postgres
// connection and astql renderer; the search store wraps the OpenSearch
// provider.
func New(db *sqlx.DB, renderer astql.Renderer, search grub.SearchProvider) *Stores {
	documents := NewDocuments(db, renderer)
	return &Stores{
		Documents: documents,
		Versions:  NewVersions(db, renderer, documents),
		Jobs:      NewJobs(db, renderer),
		Search:    NewSearch(search),
		db:        db,
	}
}
