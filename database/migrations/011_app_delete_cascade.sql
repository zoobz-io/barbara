-- +goose Up
-- An app with no release has no history, so deleting it takes everything
-- beneath it: the folder tree, the documents and their versions (already
-- cascading from documents), and the asset bookkeeping rows, which are a
-- projection of the bucket and carry nothing the bucket does not. Releases keep
-- their plain foreign key: releases are never deleted, so an app that has one is
-- refused, and that constraint stays the only one a delete can trip.
ALTER TABLE collections DROP CONSTRAINT collections_app_id_fkey;
ALTER TABLE collections ADD CONSTRAINT collections_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE documents DROP CONSTRAINT documents_app_id_fkey;
ALTER TABLE documents ADD CONSTRAINT documents_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE asset_folders DROP CONSTRAINT asset_folders_app_id_fkey;
ALTER TABLE asset_folders ADD CONSTRAINT asset_folders_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE asset_stats DROP CONSTRAINT asset_stats_app_id_fkey;
ALTER TABLE asset_stats ADD CONSTRAINT asset_stats_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE asset_stats_daily DROP CONSTRAINT asset_stats_daily_app_id_fkey;
ALTER TABLE asset_stats_daily ADD CONSTRAINT asset_stats_daily_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE asset_bookkeeping DROP CONSTRAINT asset_bookkeeping_app_id_fkey;
ALTER TABLE asset_bookkeeping ADD CONSTRAINT asset_bookkeeping_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE asset_bookkeeping DROP CONSTRAINT asset_bookkeeping_app_id_fkey;
ALTER TABLE asset_bookkeeping ADD CONSTRAINT asset_bookkeeping_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id);

ALTER TABLE asset_stats_daily DROP CONSTRAINT asset_stats_daily_app_id_fkey;
ALTER TABLE asset_stats_daily ADD CONSTRAINT asset_stats_daily_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id);

ALTER TABLE asset_stats DROP CONSTRAINT asset_stats_app_id_fkey;
ALTER TABLE asset_stats ADD CONSTRAINT asset_stats_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id);

ALTER TABLE asset_folders DROP CONSTRAINT asset_folders_app_id_fkey;
ALTER TABLE asset_folders ADD CONSTRAINT asset_folders_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id);

ALTER TABLE documents DROP CONSTRAINT documents_app_id_fkey;
ALTER TABLE documents ADD CONSTRAINT documents_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id);

ALTER TABLE collections DROP CONSTRAINT collections_app_id_fkey;
ALTER TABLE collections ADD CONSTRAINT collections_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES apps(id);
