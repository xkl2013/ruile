-- SQLite equivalent of versioned migration 001027.

CREATE TABLE IF NOT EXISTS agent_delegation_runs (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    parent_session_id TEXT NOT NULL,
    parent_message_id TEXT NOT NULL DEFAULT '',
    member_agent_id TEXT NOT NULL,
    task TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME NULL,
    finished_at DATETIME NULL,
    duration_millis INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    call_count INTEGER NOT NULL DEFAULT 0,
    base_cost_nanousd INTEGER NOT NULL DEFAULT 0,
    rated_cost_nanousd INTEGER NOT NULL DEFAULT 0,
    billed_point_micros INTEGER NOT NULL DEFAULT 0,
    usage_ledger_count INTEGER NOT NULL DEFAULT 0,
    output_preview TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_delegation_runs_tenant_created
    ON agent_delegation_runs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_delegation_runs_parent_session
    ON agent_delegation_runs (tenant_id, parent_session_id);
CREATE INDEX IF NOT EXISTS idx_agent_delegation_runs_member_agent
    ON agent_delegation_runs (tenant_id, member_agent_id);

ALTER TABLE tenant_usage_reservations ADD COLUMN delegation_run_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_reservations ADD COLUMN agent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_ledgers ADD COLUMN delegation_run_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_ledgers ADD COLUMN agent_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tenant_usage_reservations_delegation_run
    ON tenant_usage_reservations(delegation_run_id);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_delegation_run
    ON tenant_usage_ledgers(delegation_run_id);
