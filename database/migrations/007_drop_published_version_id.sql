-- +goose Up
-- Publish state now lives entirely in releases: a document is live iff the
-- app's current release carries it. The per-document published pointer is
-- retired — nothing reads or writes it after the publish/status/tags rebase.
ALTER TABLE documents DROP CONSTRAINT fk_documents_published_version;
ALTER TABLE documents DROP COLUMN published_version_id;

-- +goose Down
ALTER TABLE documents ADD COLUMN published_version_id UUID;
ALTER TABLE documents ADD CONSTRAINT fk_documents_published_version
    FOREIGN KEY (published_version_id) REFERENCES versions(id);
