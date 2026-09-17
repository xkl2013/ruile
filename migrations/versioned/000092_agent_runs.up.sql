-- Migration: 000092_agent_runs
-- Durable async execution records for service-agent jobs.

CREATE TABLE IF NOT EXISTS agent_runs (
    id                  VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id),
    user_id             VARCHAR(36) NOT NULL,
    profile_id          VARCHAR(36) NOT NULL DEFAULT '',
    run_type            VARCHAR(64) NOT NULL,
    agent_ref           VARCHAR(128) NOT NULL DEFAULT '',
    agent_version       VARCHAR(64) NOT NULL DEFAULT '',
    trigger_type        VARCHAR(64) NOT NULL DEFAULT '',
    trigger_id          VARCHAR(128) NOT NULL DEFAULT '',
    status              VARCHAR(32) NOT NULL DEFAULT 'queued',
    input               JSONB NOT NULL DEFAULT '{}'::jsonb,
    result              JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_code          VARCHAR(64) NOT NULL DEFAULT '',
    error_message       TEXT NOT NULL DEFAULT '',
    idempotency_key     VARCHAR(128) NOT NULL DEFAULT '',
    task_id             VARCHAR(160) NOT NULL DEFAULT '',
    attempt             INTEGER NOT NULL DEFAULT 0,
    queued_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at          TIMESTAMP WITH TIME ZONE,
    finished_at         TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_agent_runs_status
        CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT chk_agent_runs_type
        CHECK (run_type IN ('service_daily_report', 'service_memory_extract', 'expert_agent_test'))
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_scope_created
    ON agent_runs(tenant_id, user_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_runs_status
    ON agent_runs(status, queued_at)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_runs_trigger
    ON agent_runs(tenant_id, trigger_id)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_runs_idempotency_active
    ON agent_runs(tenant_id, user_id, run_type, idempotency_key, created_at DESC)
    WHERE status IN ('queued', 'running') AND deleted_at IS NULL;
