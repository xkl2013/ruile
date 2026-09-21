-- Expert workflow V2: resumable intake, phase tracking, step trace, and quality state.

ALTER TABLE agent_runs
    ADD COLUMN IF NOT EXISTS parent_run_id VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS requirement_snapshot_id VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phase VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS interaction JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS quality JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS resumed_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE agent_runs
    DROP CONSTRAINT IF EXISTS chk_agent_runs_status;

ALTER TABLE agent_runs
    ADD CONSTRAINT chk_agent_runs_status
    CHECK (status IN ('queued', 'running', 'waiting_input', 'succeeded', 'failed', 'cancelled'));

DROP INDEX IF EXISTS idx_agent_runs_idempotency_active;
CREATE UNIQUE INDEX idx_agent_runs_idempotency_active
    ON agent_runs(tenant_id, user_id, run_type, idempotency_key, created_at DESC)
    WHERE status IN ('queued', 'running', 'waiting_input') AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_runs_parent
    ON agent_runs(parent_run_id)
    WHERE deleted_at IS NULL AND parent_run_id <> '';
CREATE INDEX IF NOT EXISTS idx_agent_runs_phase
    ON agent_runs(status, phase, queued_at)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_run_steps (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    run_id VARCHAR(36) NOT NULL REFERENCES agent_runs(id),
    sequence INTEGER NOT NULL,
    step_type VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'running',
    model_id VARCHAR(128) NOT NULL DEFAULT '',
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CHECK (status IN ('running', 'succeeded', 'failed'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_run_steps_sequence
    ON agent_run_steps(run_id, sequence)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_run_input_revisions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    run_id VARCHAR(36) NOT NULL REFERENCES agent_runs(id),
    revision INTEGER NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'request',
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_run_input_revisions_revision
    ON agent_run_input_revisions(run_id, revision)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_requirement_snapshots (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    run_id VARCHAR(36) NOT NULL REFERENCES agent_runs(id),
    revision INTEGER NOT NULL,
    snapshot_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    assumptions JSONB NOT NULL DEFAULT '[]'::jsonb,
    missing JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_requirement_snapshots_revision
    ON agent_requirement_snapshots(run_id, revision)
    WHERE deleted_at IS NULL;
