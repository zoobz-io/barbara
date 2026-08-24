# 001 — Domain Model

Status: draft, pending review.

## What barbara is

An API service around markdown documents. Clients want markdown-driven websites;
today that markdown lives in git repos and developers commit changes. Barbara
replaces git as the source of truth: it stores the documents, versions them, and
serves everything a UI needs to edit them and everything a site needs to fetch
them. Consuming repos change from reading local md to fetching published content
from barbara's API.

## What barbara is not

- Not a site builder. Barbara knows about documents, not sites. Semantic
  grouping (which documents form a site, a nav, a section) is the user's
  business, expressed through tags.
- Not a renderer. Barbara stores and serves markdown. What consumers do with
  it — SSG builds, live rendering, anything — is out of scope.
- Not a git sync. Content moves out of git, one direction, once. There is no
  bidirectional sync and there will not be.
- Not an identity provider. Users, tenants, and sessions belong to janus.

## Entities

Four. Flat by design — the unix model. A document is a file; everything else is
convention layered on top by users.

### Document

The unit of everything. One markdown file, owned by a tenant.

- `id` — internal UUID.
- `tenant_id` — owning janus tenant. Every query is tenant-scoped.
- `key` — user-supplied identifier, unique per tenant. Path-like strings
  (`guides/install.md`) are allowed and encouraged, but barbara treats the key
  as opaque — slashes are convention, not hierarchy. Like S3, not like a
  filesystem.
- `published_version_id` — nullable pointer to the one published version.
- Timestamps.

A document holds no content. Content lives in versions.

### Version

An immutable, atomic snapshot of a document's full markdown content. No diffs,
no threads — every save is a complete version.

- `id`, `document_id`, monotonic `number` per document.
- `content` — the full markdown.
- `created_by`, `created_at`.

Invariants:

- Versions are never mutated and never deleted while their document lives.
- At most one version per document is published, tracked by the document's
  pointer. Publishing is a pointer move; so is rollback — "roll back" means
  point `published_version_id` at an older version. Nothing is copied.
- Draft state is not a flag. A document whose latest version is newer than its
  published version has unpublished work. A document with no published version
  is entirely draft.
- Every save lands a new version, so concurrent editors cannot destroy each
  other's work — worst case is two versions racing, both preserved. This
  property is load-bearing; do not "optimize" saves into in-place updates.

### Tag

User-defined semantic grouping. This is the whole answer to "what about
folders, sites, navigation, sections" — barbara does not hardcode any of them.

- Plain strings attached to documents, tenant-scoped.
- Many-to-many: a document carries any number of tags, a tag covers any number
  of documents.
- Query surface: list documents by tag.

If a customer wants a "site", they tag documents `site:docs.example.com` and
fetch by tag. If that convention hardens across every customer, we revisit
promoting it to schema. Not before.

### Asset

Binary blobs — images and whatever else markdown references. Object storage
(grub bucket, MinIO pattern), not postgres.

- `key` — unique per tenant, user-supplied, opaque.
- Same key on upload = overwrite. No versioning, by decision. An asset is an
  asset.
- Known consequence: overwrite is destructive and consuming sites will cache.
  If stale-asset pain shows up, the fix is cache headers or content-addressed
  delivery URLs — not asset versioning.

## Two planes

The same documents are consumed two ways with different auth and traffic
shapes. Both are barbara's API; neither is a rendered site.

- **Authoring** — the editing UI. Session-authenticated humans via janus,
  tenant-scoped by membership. Full read/write: documents, versions,
  publishing, tags, assets. Low traffic, consistency matters.
- **Delivery** — consuming sites and build pipelines fetching published
  content. Read-only: published versions and assets, enumeration by tag or
  full listing ("new files create new pages" requires enumeration).
  Non-interactive auth — open question below.

## Feature inventory (v1 API surface)

Documents
- Create (key), get, list, rename key, delete.
- Delete requires the document to be unpublished first — no yanking live
  content by accident. Deleting a document deletes its versions.

Versions
- Save content → creates version. List versions, get version content.
- Publish a version. Unpublish. Rollback = publish an older version.

Tags
- Add/remove tags on a document. List tags. List documents by tag.

Assets
- Upload (put by key, overwrite semantics), get, list, delete.

Delivery
- Get published content by key. Enumerate published documents, filterable by
  tag. Get asset by key.

Events (capitan, internal for v1)
- document created/deleted, version saved, published/unpublished/rolled back,
  asset uploaded/overwritten/deleted. Emitted after commit, per house rules.

## Explicit non-goals for v1

Sites as entities. Folder hierarchy. Rendering. Asset versioning. Search.
Webhooks/notifications to consumers. Git import tooling (a one-shot import
script will be wanted for migration — tooling, not domain). Editing locks or
CRDT — the version model already makes concurrent saves lossless.

## Open questions

1. **Delivery auth.** Consuming sites are services, not humans. Janus's rule is
   "validate users, not services" — service trust is mesh CA + client certs.
   Internal zoobz-io docs sites can ride the mesh; customer build pipelines
   running outside it cannot. Likely answer: mesh certs internally, tenant-
   scoped API keys for external consumers. Needs a decision before the
   delivery plane is built.
2. **Rename vs delivery.** Renaming a key breaks consumers fetching by key.
   Allow freely and let consumers cope, or leave a tombstone/redirect?
   Proposal: allow freely in v1, revisit when the first consumer complains.
3. **Version retention.** Versions immutable forever means unbounded growth.
   Fine for v1; flag for a retention policy later, not schema now.
4. **Where barbara's own docs live.** Plan documents live here in `docs/plans/`
   for now. Once barbara runs, its documentation becomes barbara documents —
   the dogfood. The migration is itself a test of the git-import tooling.
