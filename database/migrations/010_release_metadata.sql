-- +goose Up
-- Release metadata: what kind of cut a release was, what it changed, and the
-- version numbers its entries serve — everything the studio's release timeline
-- shows at a glance, written once at cut time so a release row alone answers
-- the list without joining versions or diffing entries on read.

-- kind names the operation that cut the release: a full-tree cut, a
-- single-page publish or unpublish, or a rollback. Rows that predate the
-- column are full cuts as far as anyone can tell, so that is the default.
-- source_release_id is the release a rollback copied forward;
-- subject_document_id is the page a publish or unpublish was about. Neither
-- carries a foreign key: they are provenance, not references the delete rules
-- should have to reason about (a document never released may be hard-deleted
-- even if an unpublish once named it). label is an optional note supplied
-- with the cut and, like the rest of the row, never edited afterwards.
ALTER TABLE releases ADD COLUMN kind TEXT NOT NULL DEFAULT 'cut';
ALTER TABLE releases ADD COLUMN label TEXT;
ALTER TABLE releases ADD COLUMN source_release_id UUID;
ALTER TABLE releases ADD COLUMN subject_document_id UUID;

-- The counts against the previous release: how many paths the release holds,
-- and how many were added, changed (same path, new version), removed, or
-- moved (same document, new path) since the release before it. Written in
-- the cut transaction from the same diff that drives the index projection.
ALTER TABLE releases ADD COLUMN entry_count INT NOT NULL DEFAULT 0;
ALTER TABLE releases ADD COLUMN added INT NOT NULL DEFAULT 0;
ALTER TABLE releases ADD COLUMN changed INT NOT NULL DEFAULT 0;
ALTER TABLE releases ADD COLUMN removed INT NOT NULL DEFAULT 0;
ALTER TABLE releases ADD COLUMN moved INT NOT NULL DEFAULT 0;

-- The served version's number, denormalized beside its id so a manifest reads
-- "v4" from the entry row alone. Backfilled from versions for existing rows.
ALTER TABLE release_entries ADD COLUMN version_number INT NOT NULL DEFAULT 0;

UPDATE release_entries e
SET version_number = v.version_number
FROM versions v
WHERE v.id = e.version_id;

UPDATE releases r
SET entry_count = (SELECT count(*) FROM release_entries e WHERE e.release_id = r.id);

-- The materialized diff: one row per document that differs from the previous
-- release, keyed on the document because a path can be freed and retaken in
-- one cut (one document removed at a key, another added at it). An added row
-- has no previous side; a removed row has no new side; a moved row carries
-- the previous key. Entries stay the pure manifest; this
-- table is what "what changed in #14" reads. Cascades with its release, and
-- carries no other foreign keys: every version it names is already held by
-- an entry of this release or the previous one, which is what keeps the
-- history alive. The change counts on the release row and these rows are
-- recomputable from entries (cmd/rebuild-releases) — the entries are the
-- source of truth, this is the projection.
CREATE TABLE release_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    document_id UUID NOT NULL,
    change TEXT NOT NULL,
    prev_key TEXT,
    prev_version_id UUID,
    prev_version_number INT,
    version_id UUID,
    version_number INT
);

CREATE UNIQUE INDEX idx_release_changes_release_document ON release_changes(release_id, document_id);
CREATE INDEX idx_release_changes_document_id ON release_changes(document_id);

-- +goose Down
DROP TABLE release_changes;
ALTER TABLE release_entries DROP COLUMN version_number;
ALTER TABLE releases DROP COLUMN moved;
ALTER TABLE releases DROP COLUMN removed;
ALTER TABLE releases DROP COLUMN changed;
ALTER TABLE releases DROP COLUMN added;
ALTER TABLE releases DROP COLUMN entry_count;
ALTER TABLE releases DROP COLUMN subject_document_id;
ALTER TABLE releases DROP COLUMN source_release_id;
ALTER TABLE releases DROP COLUMN label;
ALTER TABLE releases DROP COLUMN kind;
