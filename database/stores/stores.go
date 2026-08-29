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
type Stores struct {
	Documents   *Documents
	Versions    *Versions
	Apps        *Apps
	Collections *Collections
	Jobs        *Jobs
	Search      *Search
	Assets      *Assets

	db *sqlx.DB // for transactional aggregate methods (publishing)
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
	apps := NewApps(db, renderer)
	collections := NewCollections(db, renderer, documents, apps)
	// Documents materializes keys from the tree, checks the cross-table sibling
	// namespace, and consults the app's current release for the delete rules, so
	// it needs the collections and apps stores. Wired here to break the
	// construction cycle: collections already depends on documents.
	documents.collections = collections
	documents.apps = apps
	return &Stores{
		Documents:   documents,
		Versions:    versions,
		Apps:        apps,
		Collections: collections,
		Jobs:        NewJobs(db, renderer),
		Search:      NewSearch(search),
		Assets:      NewAssets(bucket),
		db:          db,
	}
}
