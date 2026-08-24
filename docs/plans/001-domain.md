# 001 — Domain and Storage Design

## What barbara is

An API service around markdown documents. Clients want markdown-driven websites;
today that markdown lives in git repos and developers commit changes. Barbara
replaces git as the source of truth: it stores documents, versions them, serves
everything a UI needs to edit them and everything a site needs to fetch them —
including full-text search over published content. Consuming repos change from
reading local md to fetching from barbara's API. Barbara will power all zoobz-io
documentation sites and customer sites alike.

## What barbara is not

- Not a site builder. Barbara knows documents, not sites. Which documents form
  a site, a nav, a section is the user's business, expressed through tags.
- Not a renderer. Barbara stores and serves markdown; what consumers do with it
  is out of scope.
- Not a git sync. Content moves out of git, one direction, once.
- Not an identity provider. Users, tenants, and sessions belong to
  [janus](https://github.com/zoobz-io/janus).
- No public consumers in the traditional sense. Barbara is an admin utility for
  sites; every consumer is either an authenticated user or a mesh service.

## Storage model

Dual storage. Postgres holds the authoring side: logical documents, immutable
versions, tags, system metadata. [OpenSearch](https://opensearch.org) holds one
document per *published*
document: the merge of the pg document (tags, system metadata) and the published
version's content. The site-facing APIs — pages and search — read OpenSearch
only, which is what makes full-text search over exclusively published content
free: the search index is the serving store, no filtering, no second lookup.

Postgres is the system of record; the OpenSearch index is always rebuildable
from it, and a full reindex job exists from day one.

## Entities

Flat by design — the unix model. A document is a file; folders, sites, and
navigation are conventions users layer on top.

### document (postgres)

The logical document. Holds identity and system metadata, no content and no
versioning.

- `id` — UUID.
- `tenant_id` — owning janus tenant. Every query is tenant-scoped.
- `key` — user-supplied identifier, unique per tenant. Path-like strings
  (`guides/install.md`) are allowed and encouraged, but barbara treats the key
  as opaque — slashes are convention, not hierarchy. S3, not a filesystem.
- `published_version_id` — nullable pointer to the one published version.
- `tags` — organizational labels, tenant-scoped strings. System-side metadata:
  how admins organize files. Distinct concern from any tags that may appear
  inside content (content metadata — travels with the document, versioned,
  out of scope until the extraction design). Stored as an array column in v1;
  promoted to its own table if tags ever need their own metadata.
- Timestamps.

### version (postgres)

An immutable, atomic snapshot of a document's full markdown content. No diffs.
Deliberately a simple table so the future extraction design can extend it —
new columns or a side table keyed by `version_id`, whichever that design wants.

- `id`, `document_id`, `tenant_id`, monotonic `version_number` per document.
- `content` — the full markdown, inline. Departure from
  [argus](https://github.com/zoobz-io/argus) (object storage
  + hash) is intentional: markdown is small text, and inline content keeps
  versions self-contained for rollback and reindex.
- `created_by`, `created_at`.

Versions are never mutated. Every save lands a new version, so concurrent
editors cannot destroy each other's work — worst case is two versions racing,
both preserved.

### asset (object storage)

Binary blobs — images and whatever else markdown references.
[Grub](https://github.com/zoobz-io/grub) bucket ([MinIO](https://min.io) dev /
S3-compatible prod), not postgres, not OpenSearch.

- `key` — unique per tenant, user-supplied, opaque.
- Same key on upload = overwrite. No versioning. An asset is an asset.

### document index (opensearch)

The projection, following argus's `DocumentVersionIndex` pattern: a dedicated
Go struct is the OS document type, indexed with the *document* ID as the OS doc
ID — one live entry per published document, replaced on update, removed on
unpublish/delete.

Merged fields: `document_id`, `tenant_id`, `key`, `tags` (keyword array),
`version_id`, `version_number`, `content` (analyzed text), timestamps. One
shared index, `tenant_id` keyword filter — not index-per-tenant. Extraction
will add fields here later via a mapping change; the mapping absorbs that
without touching postgres.

## Lifecycle

- **Save** — creates a version. Always. Never an in-place update.
- **Publish** — points `published_version_id` at a version, then writes the
  merged projection to OpenSearch. Postgres commits first; the OS write follows
  and retries until it lands (jobs pattern below), so serving can lag authoring
  by seconds. "Published" in the UI means the pointer moved.
- **Rollback** — publish an older version. Pointer move plus reindex; nothing
  is copied.
- **Unpublish** — clears the pointer, removes the OS entry.
- **Draft state is not a flag.** A document whose latest version is newer than
  its published version has unpublished work; no published version means
  entirely draft. Drafts are fully visible through the authoring API — barbara
  is an admin utility, seeing drafts is core responsibility.
- **Tag changes** on a published document update the OS entry (the projection
  includes tags) without touching the published pointer.
- **Rename** — frees the old key, old key 404s. No tombstones, no redirects.
- **Delete** — requires unpublish first; removing live content takes two
  deliberate motions. Deleting a document deletes its versions.
- **Asset overwrite** — destructive by design. Stale-cache pain, if it shows
  up, gets cache headers or content-addressed delivery URLs — not versioning.

## Auth

Services authenticate to each other with mesh CA client certificates. Barbara
wraps user requests and delegates identity and entitlement checks to janus over
the mesh (the [aegis](https://github.com/zoobz-io/aegis) gRPC surface).
Barbara has no user table, no sessions of
its own, and no API keys.

## API surface (v1)

Authoring — session-authenticated users via janus, tenant-scoped:

- Documents: create (key), get, list, rename, delete (unpublished only).
- Versions: save content, list, get.
- Publishing: publish version, unpublish, rollback (= publish older).
- Tags: add/remove on document, list documents by tag.
- Assets: put by key (overwrite), get, list, delete.

Site-facing — mesh services, reads OpenSearch only:

- Get published document by key.
- Enumerate published documents, filterable by tag ("new files create new
  pages" requires enumeration).
- Full-text search over published content, tenant-scoped.

Events ([capitan](https://github.com/zoobz-io/capitan), internal): document
created/renamed/deleted, version saved,
published/unpublished/rolled back, tags changed, asset written/deleted, index
write succeeded/failed. Emitted after commit, per house rules.

## Machinery ported from argus

- Typed OpenSearch through the house stack:
  [`grub/opensearch`](https://github.com/zoobz-io/grub) provider →
  [`sum`](https://github.com/zoobz-io/sum)`.NewSearch[DocumentIndex]` →
  [`lucene`](https://github.com/zoobz-io/lucene)`.Builder` typed queries.
  No raw OS clients, no hand-written queries.
- Index mappings as migrations: embedded JSON files under
  `database/migrations/opensearch/`, applied by boot (`EnsureIndices`).
- Tenant scoping enforced in the search store: scoped `Search` that refuses to
  run without a tenant, separate `SearchAll` for admin.
- Jobs table + [pipz](https://github.com/zoobz-io/pipz) pipeline for async OS
  writes needing retry.

## Deferred, deliberately

- **Frontmatter extraction** — gets its own design document. The version table
  and OS mapping are shaped to receive it.
- Asset versioning.
- Semantic/hybrid search (argus has the prior art when wanted; v1 is keyword).
- Webhooks, notifications, audit streams (argus DomainEvent pattern is the
  prior art).
- Version retention policy — versions are kept forever in v1.
- Git import tooling — wanted for migration, but it's a tool, not domain.
