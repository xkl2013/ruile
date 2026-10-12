-- Migration 001026: persist the queue task identifier for Skill installs.
ALTER TABLE tenant_skill_install_jobs
    ADD COLUMN IF NOT EXISTS task_id VARCHAR(160) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tenant_skill_install_jobs_task_id
    ON tenant_skill_install_jobs(task_id);
