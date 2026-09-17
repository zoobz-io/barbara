-- +goose Up
-- Bookkeeping for assets. Object storage stays the only source of truth for
-- keys and bytes; these tables are projections kept exact by every asset
-- write and delete, and rebuilt from a full listing when something drifts.
-- Nothing here is authoritative, and the studio reads every number as-of the
-- last update.

-- One row per folder path in an app's asset tree ('' is the root), with the
-- rollup of everything beneath it. A folder is a key prefix by convention, so
-- a row appears when the first key lands under it and goes away when the last
-- one leaves — unless it was created on purpose (explicit), which is how an
-- empty folder exists without a placeholder object.
CREATE TABLE asset_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    app_id UUID NOT NULL REFERENCES apps(id),
    path TEXT NOT NULL,
    count INT NOT NULL DEFAULT 0,
    bytes BIGINT NOT NULL DEFAULT 0,
    last_written_at TIMESTAMPTZ,
    explicit BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_asset_folders_app_path ON asset_folders(tenant_id, app_id, path);

-- The app-level breakdown by media family. kind '' is the all-kinds total.
-- Root only: folders never get a per-kind split.
CREATE TABLE asset_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    app_id UUID NOT NULL REFERENCES apps(id),
    kind TEXT NOT NULL,
    count INT NOT NULL DEFAULT 0,
    bytes BIGINT NOT NULL DEFAULT 0,
    last_written_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_asset_stats_app_kind ON asset_stats(tenant_id, app_id, kind);

-- Writes per UTC day, by kind ('' = all kinds), keyed on the object's
-- last-modified time. Any window ("last 7 days") is a sum over these rows at
-- read time. An overwrite moves the object to today's bucket; a delete
-- removes it from its bucket. Rows older than the longest window are pruned
-- by the rebuild.
CREATE TABLE asset_stats_daily (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    app_id UUID NOT NULL REFERENCES apps(id),
    kind TEXT NOT NULL,
    day DATE NOT NULL,
    count INT NOT NULL DEFAULT 0,
    bytes BIGINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_asset_stats_daily_app_kind_day ON asset_stats_daily(tenant_id, app_id, kind, day);

-- When each app's bookkeeping was last rebuilt from the bucket — the "as of"
-- the studio shows beside the numbers. Absent until the first rebuild.
CREATE TABLE asset_bookkeeping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    app_id UUID NOT NULL REFERENCES apps(id),
    computed_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX idx_asset_bookkeeping_app ON asset_bookkeeping(tenant_id, app_id);

-- +goose Down
DROP TABLE asset_bookkeeping;
DROP TABLE asset_stats_daily;
DROP TABLE asset_stats;
DROP TABLE asset_folders;
