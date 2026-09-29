-- Migration 001004: repair organize assignment columns after migration
-- branches were merged.
--
-- Do not fold this into an older migration. Some databases may already have
-- recorded that migration as applied while still missing these columns.

ALTER TABLE organize_configs
    ADD COLUMN IF NOT EXISTS target_service_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE organize_jobs
    ADD COLUMN IF NOT EXISTS target_service_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS assigned_service_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS assignment_status VARCHAR(32) NOT NULL DEFAULT 'pending';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS assignment_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_organize_configs_target_service
    ON organize_configs(tenant_id, target_service_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_organize_jobs_target_service
    ON organize_jobs(tenant_id, target_service_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_organize_outputs_assignment
    ON organize_outputs(tenant_id, user_id, assignment_status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_organize_outputs_assigned_service
    ON organize_outputs(tenant_id, assigned_service_id, updated_at DESC);

UPDATE organize_outputs
SET assignment_status = 'pending',
    assignment_reason = '历史整理结果，等待用户分配'
WHERE assignment_status = '' OR assignment_status IS NULL;
