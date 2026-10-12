-- SQLite equivalent of versioned migration 001026.
ALTER TABLE tenant_skill_install_jobs
    ADD COLUMN task_id VARCHAR(160) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tenant_skill_install_jobs_task_id
    ON tenant_skill_install_jobs(task_id);
