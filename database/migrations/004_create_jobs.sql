-- +goose Up
-- Outbox for async OpenSearch writes. Publishing commits Postgres first and
-- enqueues the projection write here inside the same transaction (so the pointer
-- move and the job land atomically); the pipeline claims pending jobs and writes
-- to OpenSearch, retrying until it lands. Serving can lag authoring by seconds.
--
-- No FK on document_id on purpose: an OS-delete job must survive deletion of the
-- document row so the projection is actually removed.
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    document_id UUID NOT NULL,
    operation TEXT NOT NULL,            -- 'index' | 'delete'
    payload JSONB,                      -- the projection for 'index'; null for 'delete'
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'processing' | 'done' | 'failed'
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Claim pending work; scope by tenant and document for lookups.
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_document_id ON jobs(document_id);
CREATE INDEX idx_jobs_tenant_id ON jobs(tenant_id);

-- +goose Down
DROP TABLE jobs;
