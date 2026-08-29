-- +goose Up
-- Apps, collections, and releases: the tree and the release model. The new
-- document columns start nullable and loosely indexed here, so this migration
-- could land ahead of the code that populates them; later migrations retire
-- the published pointer (007) and tighten the tree columns to their final
-- shape (008).

-- The release unit. A tenant owns many apps; an app owns a collection tree and
-- a linear release history. current_release_id is the only mutable pointer in
-- the system — its history is the releases table. The FK is added after the
-- releases table exists (apps <-> releases cycle, broken by nullability).
CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    current_release_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_apps_tenant_name ON apps(tenant_id, name);

-- A folder: identity, name, parent — nothing else. No child counts, no
-- rollups; a collection never learns about its contents. parent_id NULL means
-- the app root. app_id is denormalized onto every row so scoping never walks
-- to the root.
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    app_id UUID NOT NULL REFERENCES apps(id),
    parent_id UUID REFERENCES collections(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sibling names are unique including at the root: NULLS NOT DISTINCT (pg15+)
-- makes two root collections with the same name collide even though their
-- parent_id is NULL. The full sibling namespace also spans documents (a doc
-- and a collection may not share a parent and name) — that half cannot be an
-- index across two tables and is enforced in the store transaction.
CREATE UNIQUE INDEX idx_collections_app_parent_name
    ON collections(app_id, parent_id, name) NULLS NOT DISTINCT;
CREATE INDEX idx_collections_parent_id ON collections(parent_id);
CREATE INDEX idx_collections_tenant_id ON collections(tenant_id);

-- An immutable snapshot of everything live in an app. Append-only: never
-- mutated, never deleted. number is monotonic per app; serializing cuts is
-- the store's job (app-row lock), and the unique index is the schema backstop
-- that turns a racing cut into a retryable error.
CREATE TABLE releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id),
    tenant_id UUID NOT NULL,
    number INT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_releases_app_number ON releases(app_id, number);
CREATE INDEX idx_releases_tenant_id ON releases(tenant_id);

-- One row per live path in a release — the materialized tree. RESTRICT on
-- document_id and version_id is the schema-enforced half of "delete frees the
-- namespace, not the history": a document or version referenced by any release
-- cannot be hard-deleted, even through the versions cascade on document
-- delete. The store checks before deleting; the FK refuses if the check races
-- a concurrent cut. The surrogate id exists for the store machinery; the
-- unique (release_id, key) is the real identity.
CREATE TABLE release_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    version_id UUID NOT NULL REFERENCES versions(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_release_entries_release_key ON release_entries(release_id, key);
CREATE INDEX idx_release_entries_document_id ON release_entries(document_id);
CREATE INDEX idx_release_entries_version_id ON release_entries(version_id);

-- Close the apps <-> releases cycle now that both tables exist.
ALTER TABLE apps ADD CONSTRAINT fk_apps_current_release
    FOREIGN KEY (current_release_id) REFERENCES releases(id);

-- Tree placement for documents. collection_id NULL means the app root and
-- stays nullable forever; app_id and name start nullable (tightened in 008).
-- deleted_at is the soft-delete marker for documents referenced by a
-- historical release (delete frees the key and name via the partial indexes;
-- versions survive).
ALTER TABLE documents ADD COLUMN app_id UUID REFERENCES apps(id);
ALTER TABLE documents ADD COLUMN collection_id UUID REFERENCES collections(id);
ALTER TABLE documents ADD COLUMN name TEXT;
ALTER TABLE documents ADD COLUMN deleted_at TIMESTAMPTZ;

CREATE INDEX idx_documents_app_id ON documents(app_id);
CREATE INDEX idx_documents_collection_id ON documents(collection_id);

-- +goose Down
DROP INDEX idx_documents_collection_id;
DROP INDEX idx_documents_app_id;
ALTER TABLE documents DROP COLUMN deleted_at;
ALTER TABLE documents DROP COLUMN name;
ALTER TABLE documents DROP COLUMN collection_id;
ALTER TABLE documents DROP COLUMN app_id;
ALTER TABLE apps DROP CONSTRAINT fk_apps_current_release;
DROP TABLE release_entries;
DROP TABLE releases;
DROP TABLE collections;
DROP TABLE apps;
