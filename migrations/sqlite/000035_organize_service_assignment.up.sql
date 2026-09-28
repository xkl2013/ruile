-- Migration 000035: bind organize configurations and outputs to service spaces.

ALTER TABLE organize_configs
    ADD COLUMN target_service_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE organize_jobs
    ADD COLUMN target_service_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN assigned_service_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN assignment_status VARCHAR(32) NOT NULL DEFAULT 'pending';

ALTER TABLE organize_outputs
    ADD COLUMN assignment_reason TEXT NOT NULL DEFAULT '';

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
