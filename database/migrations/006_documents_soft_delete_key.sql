-- +goose Up
-- Free a soft-deleted document's key. Plan 002 delete semantics: a document
-- referenced by a historical release is soft-deleted (deleted_at set, versions
-- kept) rather than removed, and its key must become available for a new
-- document at the same path. Making the per-tenant key index partial on
-- deleted_at IS NULL stops a tombstoned row from holding its key.
--
-- Scope stays per-tenant here, matching migration 002; the per-app key/name
-- tightening is a later migration (see 005), once every write path populates
-- the tree columns.
DROP INDEX idx_documents_tenant_key;
CREATE UNIQUE INDEX idx_documents_tenant_key
    ON documents(tenant_id, key)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX idx_documents_tenant_key;
CREATE UNIQUE INDEX idx_documents_tenant_key ON documents(tenant_id, key);
