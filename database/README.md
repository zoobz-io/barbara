# database

The data layer, in three passes. [`models/`](models/) holds the domain types — plain
Go structs, one per table. [`stores/`](stores/) is the Postgres access layer over those
types. [`migrations/`](migrations/) is the schema itself: SQL files that create the
tables the models map to and the indexes the stores rely on.

A store reads and writes a model; a model maps 1:1 to a table a migration created.

Stores are shared across all API surfaces — construct them once (in
`internal/boot`), register a narrow per-surface contract over each. Multi-store
writes whose invariants require atomicity live only as transactional methods on
the stores aggregate, never composed from individual store calls at call sites.
