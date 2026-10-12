DROP INDEX IF EXISTS idx_tenant_skill_install_jobs_task_id;
ALTER TABLE tenant_skill_install_jobs
    DROP COLUMN IF EXISTS task_id;
