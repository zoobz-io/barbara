# reindex

Full-reindex command entrypoint.

## Purpose

A one-shot operational command that rebuilds the OpenSearch index from Postgres,
the system of record. Run it when the index is lost, corrupted, or a mapping
change needs a rebuild: it walks every published document across all tenants and
re-projects each into OpenSearch, idempotently.

It is a **command, not an HTTP endpoint** — the reindex is tenant-agnostic
operational tooling, and a cross-tenant rebuild has no place on the per-tenant
request surface (`api`/`admin`).

## Behavior

- Enumerates all published documents (every tenant), keyset-paged.
- Builds each projection with the same logic as publish
  (`database/transformers.Projection`) — no divergent second code path.
- Writes each projection through the search store's write side, upserting by
  document id, so re-running converges to one live entry per published document.
- Repopulates rather than reconciles: clearing entries for no-longer-published
  documents is the job of dropping the index (a mapping-change rebuild recreates
  it empty first).

## Run

```sh
go run ./cmd/reindex
```

Uses the same shared config as the `api`/`admin` binaries (Postgres, OpenSearch,
storage). Logs the number of documents projected and exits.
