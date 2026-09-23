-- Migration 000102 rollback.
DROP TABLE IF EXISTS agent_requirement_snapshots;
DROP TABLE IF EXISTS agent_run_input_revisions;
DROP TABLE IF EXISTS agent_run_steps;

DROP INDEX IF EXISTS idx_agent_runs_phase;
DROP INDEX IF EXISTS idx_agent_runs_parent;
DROP INDEX IF EXISTS idx_agent_runs_idempotency_active;
CREATE UNIQUE INDEX idx_agent_runs_idempotency_active
    ON agent_runs(tenant_id, user_id, run_type, idempotency_key, created_at DESC)
    WHERE status IN ('queued', 'running') AND deleted_at IS NULL;

ALTER TABLE agent_runs
    DROP CONSTRAINT IF EXISTS chk_agent_runs_status;
ALTER TABLE agent_runs
    ADD CONSTRAINT chk_agent_runs_status
    CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled'));

ALTER TABLE agent_runs
    DROP COLUMN IF EXISTS resumed_at,
    DROP COLUMN IF EXISTS quality,
    DROP COLUMN IF EXISTS interaction,
    DROP COLUMN IF EXISTS phase,
    DROP COLUMN IF EXISTS requirement_snapshot_id,
    DROP COLUMN IF EXISTS parent_run_id;
