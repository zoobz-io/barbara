# 002 — Collections, Apps, and Releases

## Why this exists

001 shipped a flat, per-document world: opaque path-like keys, a mutable
published pointer per document. Three problems surfaced once the consumer
surface got real:

- **Structure.** Users manage documents in folders. Prefix-convention keys give
  no real hierarchy: no empty folders, no cheap folder rename, no folder
  listing without string games.
- **Atomicity.** Per-document publish cannot update five pages and a
  restructure together; intermediate states are visible on the live site.
  Moving a published document mutates a live URL mid-edit.
- **Audit.** The published pointer is a destructive overwrite. Versions record
  what the content *was*; nothing records what was *live when*, or who changed
  that.

One model change resolves all three, and it is git's model wearing SQL: version
= blob, collection = tree, release = commit. Barbara replaces git; it steals
git's one great idea and declines its implementation (see non-goals).

## Entities

### app (postgres, new)

The release unit. A tenant has many apps; an app owns a collection tree and a
linear release history. "Multiple repos per customer" is the shape being
replaced, and this is its successor.

- `id`, `tenant_id`, `name` (unique per tenant).
- `current_release_id` — nullable pointer to the live release. **The only
  mutable pointer left in the system.** Its history is the releases table.
- Timestamps. No other metadata yet — columns get added when real requirements
  name them, not speculatively.

### collection (postgres, new)

A folder. The dentry model: identity, name, parent — nothing else. A
collection never learns anything about its contents; no child counts, no
aggregate status, no rollups. Anything a UI wants beyond the listing is
answered at read time by a query or the index.

- `id`, `tenant_id`, `app_id`, `parent_id` (null = app root), `name`.
- Unique `(app_id, parent_id, name)`. `app_id` is denormalized onto every row
  so scoping never walks to the root.
- Empty collections exist (a thing prefix keys never allowed).
- Delete refuses unless empty — rmdir, not rm -rf.

### document (postgres, changed)

Gains its place in the tree; stays the identity-and-metadata row.

- New: `app_id`, `collection_id` (null = app root), `name` (unique per
  `(app_id, collection_id)` shared with sibling collections).
- `key` survives as the **materialized full path** — a cache, derived from the
  tree, rewritten in-transaction when an ancestor moves or renames. Uniqueness
  moves from per-tenant to per-app. Every existing key-based query, the index
  mapping, and lookup-by-key survive untouched. Moves are rare; reads are
  constant; pay on the rare side.
- Dropped: `published_version_id`. Publish state now lives in releases; a
  document is "published" iff the app's current release carries it. The
  derived draft/published/published-with-newer-draft status compares against
  the current release entry instead of a pointer.
- New: `deleted_at` (nullable). See delete semantics.

### version (postgres, unchanged)

Already immutable, already attributed (`created_by`). Releases reference
versions; that reference is what makes history survivable (below).

### release (postgres, new)

An immutable snapshot of everything live in an app. Append-only; never
mutated, never deleted.

- `id`, `app_id`, `tenant_id`, `number` (monotonic per app), `created_by`,
  `created_at`.
- `release_entries`: `(release_id, key, document_id, version_id)` — the full
  materialized tree, one row per live path. No delta chains: every historical
  release is directly queryable, and "what was live last Tuesday and who cut
  it" is a table scan. Markdown-scale trees make full materialization cheap
  rows; the storage is the audit trail.

### document index (opensearch, changed)

Still one live entry per published document — now meaning: per entry in the
app's current release. Mapping gains `app_id` (keyword) and `parent_path`
(keyword) alongside the existing fields, materialized at projection time.
Folder listing on the live site is one term query on `parent_path`; page fetch
stays one term query on `key`. The tree is flattened into the index at release
time — no tree-walking at read time, ever.

## Lifecycle

- **Save** — unchanged. New version, never in-place. (Optimistic concurrency
  via `base_version` is ticket #50, orthogonal to this plan.)
- **Release** — the only publish primitive. Cutting a release writes the
  release row and its entries in one transaction, moves `current_release_id`,
  and enqueues projection jobs for the *diff* against the previous release:
  changed/added paths upsert into OpenSearch, removed paths delete. Postgres
  commits first; serving lags by seconds, as today.
- **Publish / unpublish a document** — sugar over release. "Publish this doc"
  cuts a release of current-tree-plus-this-document's-head. "Unpublish" cuts
  one without the path. One mechanism underneath; two publishing systems side
  by side is a non-goal enforced with prejudice.
- **Rollback** — cuts a *new* release copying an old release's entries. The
  pointer never moves backward; release numbers stay a straight line and the
  audit log needs no asterisk. Git's revert, not git's reset.
- **Move / rename** (documents and collections) — authoring-only mutations.
  Nothing live changes until the next release, so there is no "moving a live
  page" case: no partial states, no mid-edit 404s. Old paths are preserved in
  old releases' entries, which resolves 001's rename-loses-history gap.
  Renaming a collection rewrites descendant `key`s in the same transaction.
  No redirects for moved paths; the old path 404s on the next release
  (redirects are deferred, below).
- **Delete frees the namespace, not the history.** A document absent from the
  current release is unpublished and may be deleted:
  - Never referenced by any release → hard delete, versions cascade (nothing
    ever pointed at them).
  - Referenced by any historical release → soft delete: `deleted_at` set, key
    freed (partial unique index `WHERE deleted_at IS NULL`), versions
    retained. Old releases stay intact; rollback never hits a hole. Deleting
    a file from a repo does not delete blobs from history.
- **Draft state is not a flag** — unchanged in spirit; the comparison target
  is now the current release entry.
- **Tags** — unchanged semantics; the tag re-projection path keys off the
  current release entry instead of the published pointer.
- **Reindex** — rebuilds from each app's current release. Same job, new
  source query.

## API surface (delta over 001, post-#46 layout)

All tenant-scoped; app-scoped routes carry the app id.

Authoring:

- Apps: create (name), get, list, rename, delete (only with no releases —
  releases are never deleted).
- Collections: create (in parent), rename, move, delete (empty only), list
  contents — one round trip, subcollections and documents together with
  derived status.
- Documents: as today, plus move (new parent, new name); create takes a
  collection.
- Releases: cut (full-tree snapshot of head versions, or explicit entries),
  list, get (with entries), rollback (copy an old release forward).
- Document publish/unpublish sugar endpoints remain, reimplemented over
  releases.

Site-facing (mesh, OpenSearch only, scoped by tenant + app):

- Get published document by key, enumerate (tag filter), full-text search —
  as today, plus app scoping and folder listing by `parent_path`.

Scopes (#47) apply: release-cutting sits behind the publish scope.

Events: app created/renamed/deleted, collection created/renamed/moved/deleted,
document moved, release cut, rollback — joining the existing set.

## Sequencing

Tickets #46–#51 (surface fold, scopes, head-content, status, save
concurrency, admin stub) land first — none are invalidated, all get app/
collection scoping folded in as this plan's migrations arrive. This plan is
migrations 005+ plus store/handler work on top of the folded surface.

Migration of existing data: none — no deployed environment exists, so there
is no data to carry across. (An earlier revision planned a one-shot backfill
seeding the tree from path-like keys; it was built and then withdrawn when
this was pointed out.) The schema consequence survives for a different
reason: the new document columns stay nullable until every write path
populates them, then a follow-up migration tightens NOT NULL, swaps the key
uniqueness from per-tenant to per-app, and adds the namespace indexes — that
tightening sequences on code landing, not on any backfill run.

## Assets (amended 2026-08-29)

Assets stay what 001 made them: opaque blobs in the bucket, no Postgres row,
no versioning, same-key overwrite. Two changes and three explicit
non-decisions-made-decisions:

- **App-scoped namespace.** The stored object key becomes
  `<tenant>/<app>/<key>`; the user key is unique per app, and listings filter
  by the app prefix. The one-off backfill moves existing objects from
  `<tenant>/<key>` into the tenant's seeded app.
- **Site-facing asset read.** The live site needs bytes for the images its
  markdown references. The mesh surface gains a get-asset-by-key read, served
  straight from the bucket — assets are not in OpenSearch and not in releases,
  so "current" is the only version there is.
- **A folder is a key prefix, not a collection.** Assets do not join the tree:
  no tree entry, no name collision with documents or collections, and a
  collection rename does not move assets sharing the prefix. Markdown
  references may go stale when assets move or change; accepted deliberately.
- **Releases do not cover assets.** A rollback restores last week's markdown
  with today's images. Accepted deliberately — the version game is not played
  with binaries.

## Non-goals

- **No branching, no merging, no three-way anything.** Linear release history
  per app. The day a merge algorithm shows up in a design doc, this section is
  the veto.
- **No git implementation.** The model is git's; the implementation is
  Postgres + OpenSearch already in the stack. Git enters, if ever, as a sync
  boundary at the edge (import/export), never as the core.
- **No eager rollups on collections.** The year-of-consistency-bugs door
  stays shut.

## Deferred, deliberately

- App metadata beyond name (serving slug, domain, theme) — columns when
  requirements name them.
- Redirects for moved/renamed paths — old releases hold the data; a redirect
  layer can be derived later without schema work.
- Release labels/notes, retention policy — releases are kept forever in v1,
  same stance as versions.
- Collection metadata (titles, ordering for nav) — pure namespace until a
  real nav requirement lands.
- Everything 001 already deferred (frontmatter extraction, asset versioning,
  semantic search, webhooks, git import tooling).
