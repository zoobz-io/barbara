# rebuild-assets

Asset-bookkeeping rebuild command entrypoint.

## Purpose

A one-shot operational command that rebuilds every app's asset bookkeeping —
the folder rollups, the per-kind stats, and the daily write series behind the
studio's folder browser and asset stats — from object storage, the source of
truth for assets. Run it to backfill apps whose objects landed before the
bookkeeping tables existed, or to repair rows that drifted after a
bookkeeping failure (`barbara.asset.bookkeeping.failed`).

It is a **command, not an HTTP endpoint** — the rebuild is tenant-agnostic
operational tooling, and a cross-tenant rebuild has no place on the per-tenant
request surface (`api`/`admin`).

## Behavior

- Enumerates every app (every tenant), keyset-paged.
- Lists each app's objects once, recursively, and folds the listing into the
  rows the tables should hold — the same fold the store's own `Rebuild` uses.
- Replaces the app's rows in one transaction: derived folder rows are
  rewritten, explicit (created-on-purpose) folders keep their flag, the kind
  and day tables are rewritten whole, and the app's `computed_at` stamp is
  set — the "as of" the studio can show beside the numbers.
- Idempotent: re-running converges on the listing. On an error it reports the
  apps rebuilt so far, so a re-run resumes safely.

## Run

```sh
go run ./cmd/rebuild-assets
make rebuild-assets
```

Uses the same shared config as the `api`/`admin` binaries (Postgres, object
storage, OpenSearch). Logs the number of apps rebuilt and exits.
