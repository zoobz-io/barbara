-- +goose Up
-- The logical document: identity and system metadata, no content, no versioning.
-- tenant_id is the owning janus tenant — barbara is not an identity provider, so
-- there is no tenants table to reference; every query is tenant-scoped in code.
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    key TEXT NOT NULL,
    published_version_id UUID,
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- key is user-supplied and unique per tenant. Path-like strings are allowed but
-- opaque — slashes are convention, not hierarchy. The composite unique also
-- serves tenant-scoped listing (tenant_id is the leftmost column).
CREATE UNIQUE INDEX idx_documents_tenant_key ON documents(tenant_id, key);

-- Tag lookup (list documents by tag) over the array column.
CREATE INDEX idx_documents_tags ON documents USING GIN(tags);

-- +goose Down
DROP TABLE documents;
