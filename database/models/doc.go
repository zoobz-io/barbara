// Package models holds barbara's domain types — plain Go structs, one per
// table (postgres) or index (opensearch). Models are the source of truth for
// entity structure; a store reads and writes a model, and a model maps 1:1 to
// the schema a migration created.
package models
