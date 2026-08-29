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

Migration of existing data: current documents' path-like keys can seed
collections mechanically (split on slash, create the tree, attach documents);
each tenant's flat corpus becomes one app. A one-shot migration, part of this
plan's first ticket.

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
