ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS job_mode VARCHAR(16) NOT NULL DEFAULT 'single';
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS selection_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS input_fingerprint VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS selected_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS ready_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS processed_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS failed_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS overlap_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS batch_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN IF NOT EXISTS coverage JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_organize_jobs_input_fingerprint
    ON organize_jobs(tenant_id, user_id, input_fingerprint, created_at DESC)
    WHERE deleted_at IS NULL AND input_fingerprint <> '';

CREATE TABLE IF NOT EXISTS organize_job_batches (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    parent_job_id VARCHAR(36) NOT NULL,
    batch_index INTEGER NOT NULL,
    batch_count INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    stage VARCHAR(64) NOT NULL DEFAULT 'queued',
    progress INTEGER NOT NULL DEFAULT 0,
    memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    input_chars INTEGER NOT NULL DEFAULT 0,
    prompt_hash VARCHAR(64) NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    structured_result JSONB NOT NULL DEFAULT '{}'::jsonb,
    citations JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_job_batches_status
        CHECK (status IN ('queued', 'running', 'completed', 'fallback', 'failed', 'canceled')),
    CONSTRAINT chk_organize_job_batches_progress
        CHECK (progress >= 0 AND progress <= 100)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_job_batches_parent_index
    ON organize_job_batches(parent_job_id, batch_index)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_organize_job_batches_scope
    ON organize_job_batches(tenant_id, user_id, parent_job_id, batch_index)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_organize_job_batches_status
    ON organize_job_batches(status, updated_at DESC)
    WHERE deleted_at IS NULL;
