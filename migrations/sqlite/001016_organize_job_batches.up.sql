ALTER TABLE organize_jobs ADD COLUMN job_mode TEXT NOT NULL DEFAULT 'single';
ALTER TABLE organize_jobs ADD COLUMN selection_snapshot TEXT NOT NULL DEFAULT '{}';
ALTER TABLE organize_jobs ADD COLUMN input_fingerprint TEXT NOT NULL DEFAULT '';
ALTER TABLE organize_jobs ADD COLUMN selected_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN ready_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN processed_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN failed_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN overlap_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN batch_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_jobs ADD COLUMN coverage TEXT NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_organize_jobs_input_fingerprint
    ON organize_jobs(tenant_id, user_id, input_fingerprint, created_at DESC);

CREATE TABLE IF NOT EXISTS organize_job_batches (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    parent_job_id TEXT NOT NULL,
    batch_index INTEGER NOT NULL,
    batch_count INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    stage TEXT NOT NULL DEFAULT 'queued',
    progress INTEGER NOT NULL DEFAULT 0,
    memory_ids TEXT NOT NULL DEFAULT '[]',
    input_chars INTEGER NOT NULL DEFAULT 0,
    prompt_hash TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    structured_result TEXT NOT NULL DEFAULT '{}',
    citations TEXT NOT NULL DEFAULT '{}',
    error_message TEXT NOT NULL DEFAULT '',
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (status IN ('queued', 'running', 'completed', 'fallback', 'failed', 'canceled')),
    CHECK (progress >= 0 AND progress <= 100)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_job_batches_parent_index
    ON organize_job_batches(parent_job_id, batch_index)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_organize_job_batches_scope
    ON organize_job_batches(tenant_id, user_id, parent_job_id, batch_index);

CREATE INDEX IF NOT EXISTS idx_organize_job_batches_status
    ON organize_job_batches(status, updated_at DESC);
