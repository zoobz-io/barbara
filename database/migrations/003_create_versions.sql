-- +goose Up
-- An immutable, atomic snapshot of a document's full markdown. No diffs, never
-- mutated. Content is stored inline (a deliberate departure from argus's object
-- storage + hash): markdown is small text, and inline content keeps versions
-- self-contained for rollback and reindex.
CREATE TABLE versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    version_number INT NOT NULL,
    content TEXT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- version_number is monotonic per document; two concurrent saves race to distinct
-- numbers, both preserved.
CREATE UNIQUE INDEX idx_versions_document_version ON versions(document_id, version_number);
CREATE INDEX idx_versions_tenant_id ON versions(tenant_id);

-- Resolve the documents <-> versions chicken/egg: documents.published_version_id
-- was created without its FK (nullable pointer), now that versions exists it can
-- reference it. One published version per document; the pointer moves on
-- publish/rollback and is cleared on unpublish.
ALTER TABLE documents ADD CONSTRAINT fk_documents_published_version
    FOREIGN KEY (published_version_id) REFERENCES versions(id);

-- +goose Down
ALTER TABLE documents DROP CONSTRAINT IF EXISTS fk_documents_published_version;
DROP TABLE versions;
