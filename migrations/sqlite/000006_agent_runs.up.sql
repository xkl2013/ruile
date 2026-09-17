-- SQLite equivalent of versioned migration 000089.

CREATE TABLE IF NOT EXISTS agent_runs (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    user_id TEXT NOT NULL,
    profile_id TEXT NOT NULL DEFAULT '',
    run_type TEXT NOT NULL,
    agent_ref TEXT NOT NULL DEFAULT '',
    agent_version TEXT NOT NULL DEFAULT '',
    trigger_type TEXT NOT NULL DEFAULT '',
    trigger_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued',
    input TEXT NOT NULL DEFAULT '{}',
    result TEXT NOT NULL DEFAULT '{}',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    attempt INTEGER NOT NULL DEFAULT 0,
    queued_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    CHECK (run_type IN ('service_daily_report', 'service_memory_extract', 'expert_agent_test'))
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_scope_created
    ON agent_runs(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_status
    ON agent_runs(status, queued_at);
CREATE INDEX IF NOT EXISTS idx_agent_runs_trigger
    ON agent_runs(tenant_id, trigger_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_runs_idempotency_active
    ON agent_runs(tenant_id, user_id, run_type, idempotency_key, created_at DESC);
