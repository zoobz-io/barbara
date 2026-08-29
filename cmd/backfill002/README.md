# backfill002

One-shot plan-002 backfill. Seeds the tree from existing path-like document
keys — one `default` app per tenant, a collection per key prefix, tree
placement on every document — and cuts release 1 per app from the
`published_version_id` pointers, so existing publish state survives the
pointer's removal.

Run it once per environment, after migration 005 applies and before the
tightening migration (NOT NULLs, per-app namespace indexes) ships:

    go run ./cmd/backfill002

Idempotent: a tenant that already has an app is skipped, so rerunning adds no
duplicate rows. One transaction per tenant — a failure leaves that tenant
untouched and rerunning picks it up.

No domain events, no projection jobs: the backfill replays no domain activity,
and the OpenSearch index already matches the pointers release 1 snapshots.

Keys are unchanged. Empty key segments (doubled or leading slashes) are
dropped when deriving the tree; a degenerate all-slash key lands at the app
root under its verbatim key.
