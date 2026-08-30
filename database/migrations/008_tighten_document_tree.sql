-- +goose Up
-- Tighten the document tree to its final shape. Every write path populates the
-- tree columns, so the loose interim state has no reason to exist: placement
-- becomes mandatory, and uniqueness scopes to the app — the serving unit —
-- rather than the tenant.
ALTER TABLE documents ALTER COLUMN app_id SET NOT NULL;
ALTER TABLE documents ALTER COLUMN name SET NOT NULL;

-- Key uniqueness moves from per-tenant to per-app: a key names a page within
-- one app, and two apps of one tenant may hold the same key. Partial on
-- deleted_at so a soft-deleted document (tombstoned because a historical
-- release references it) frees its key.
DROP INDEX idx_documents_tenant_key;
CREATE UNIQUE INDEX idx_documents_app_key
    ON documents(app_id, key)
    WHERE deleted_at IS NULL;

-- Sibling document names are unique per (app, collection), including at the
-- app root: NULLS NOT DISTINCT makes two root documents with the same name
-- collide despite NULL collection_id — the same treatment collections got.
-- Partial on deleted_at so soft delete frees the name as well as the key. The
-- cross-table half of the namespace (a document and a collection may not share
-- a parent and name) is enforced in the store transaction; an index cannot
-- span two tables.
CREATE UNIQUE INDEX idx_documents_app_collection_name
    ON documents(app_id, collection_id, name) NULLS NOT DISTINCT
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX idx_documents_app_collection_name;
DROP INDEX idx_documents_app_key;
CREATE UNIQUE INDEX idx_documents_tenant_key
    ON documents(tenant_id, key)
    WHERE deleted_at IS NULL;
ALTER TABLE documents ALTER COLUMN name DROP NOT NULL;
ALTER TABLE documents ALTER COLUMN app_id DROP NOT NULL;
