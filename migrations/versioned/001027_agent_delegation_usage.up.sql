-- Migration 001027: durable member Agent delegation records and billing attribution.

CREATE TABLE IF NOT EXISTS agent_delegation_runs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    parent_session_id VARCHAR(128) NOT NULL,
    parent_message_id VARCHAR(128) NOT NULL DEFAULT '',
    member_agent_id VARCHAR(128) NOT NULL,
    task TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at TIMESTAMP NULL,
    finished_at TIMESTAMP NULL,
    duration_millis BIGINT NOT NULL DEFAULT 0,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    cached_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    call_count BIGINT NOT NULL DEFAULT 0,
    base_cost_nanousd BIGINT NOT NULL DEFAULT 0,
    rated_cost_nanousd BIGINT NOT NULL DEFAULT 0,
    billed_point_micros BIGINT NOT NULL DEFAULT 0,
    usage_ledger_count INTEGER NOT NULL DEFAULT 0,
    output_preview TEXT NOT NULL DEFAULT '',
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_delegation_runs_tenant_created
    ON agent_delegation_runs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_delegation_runs_parent_session
    ON agent_delegation_runs (tenant_id, parent_session_id);
CREATE INDEX IF NOT EXISTS idx_agent_delegation_runs_member_agent
    ON agent_delegation_runs (tenant_id, member_agent_id);

ALTER TABLE tenant_usage_reservations
    ADD COLUMN IF NOT EXISTS delegation_run_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_reservations
    ADD COLUMN IF NOT EXISTS agent_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_ledgers
    ADD COLUMN IF NOT EXISTS delegation_run_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_ledgers
    ADD COLUMN IF NOT EXISTS agent_id VARCHAR(128) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tenant_usage_reservations_delegation_run
    ON tenant_usage_reservations(delegation_run_id);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_delegation_run
    ON tenant_usage_ledgers(delegation_run_id);
