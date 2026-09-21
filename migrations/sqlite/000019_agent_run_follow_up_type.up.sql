-- Rebuild upgraded Lite AgentRun tables so explanation follow-ups are
-- accepted while preserving workflow and thread state.

CREATE TABLE agent_runs_new (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    user_id TEXT NOT NULL,
    profile_id TEXT NOT NULL DEFAULT '',
    parent_run_id TEXT NOT NULL DEFAULT '',
    requirement_snapshot_id TEXT NOT NULL DEFAULT '',
    run_type TEXT NOT NULL,
    agent_ref TEXT NOT NULL DEFAULT '',
    agent_version TEXT NOT NULL DEFAULT '',
    trigger_type TEXT NOT NULL DEFAULT '',
    trigger_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued',
    phase TEXT NOT NULL DEFAULT '',
    input TEXT NOT NULL DEFAULT '{}',
    interaction TEXT NOT NULL DEFAULT '{}',
    quality TEXT NOT NULL DEFAULT '{}',
    result TEXT NOT NULL DEFAULT '{}',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    attempt INTEGER NOT NULL DEFAULT 0,
    queued_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resumed_at DATETIME,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    thread_id TEXT NOT NULL DEFAULT '',
    CHECK (status IN ('queued', 'running', 'waiting_input', 'succeeded', 'failed', 'cancelled')),
    CHECK (run_type IN ('service_daily_report', 'service_memory_extract', 'expert_agent_test', 'expert_follow_up'))
);

INSERT INTO agent_runs_new (
    id, tenant_id, user_id, profile_id, parent_run_id,
    requirement_snapshot_id, run_type, agent_ref, agent_version,
    trigger_type, trigger_id, status, phase, input, interaction, quality,
    result, error_code, error_message, idempotency_key, task_id, attempt,
    queued_at, resumed_at, started_at, finished_at, created_at, updated_at,
    deleted_at, thread_id
)
SELECT
    id, tenant_id, user_id, profile_id, parent_run_id,
    requirement_snapshot_id, run_type, agent_ref, agent_version,
    trigger_type, trigger_id, status, phase, input, interaction, quality,
    result, error_code, error_message, idempotency_key, task_id, attempt,
    queued_at, resumed_at, started_at, finished_at, created_at, updated_at,
    deleted_at, CASE WHEN thread_id = '' THEN id ELSE thread_id END
FROM agent_runs;

DROP TABLE agent_runs;
ALTER TABLE agent_runs_new RENAME TO agent_runs;

CREATE INDEX idx_agent_runs_scope_created
    ON agent_runs(tenant_id, user_id, created_at DESC);
CREATE INDEX idx_agent_runs_status
    ON agent_runs(status, queued_at);
CREATE INDEX idx_agent_runs_trigger
    ON agent_runs(tenant_id, trigger_id);
CREATE INDEX idx_agent_runs_parent
    ON agent_runs(parent_run_id);
CREATE INDEX idx_agent_runs_phase
    ON agent_runs(status, phase, queued_at);
CREATE INDEX idx_agent_runs_thread
    ON agent_runs(tenant_id, user_id, thread_id, created_at);
CREATE UNIQUE INDEX idx_agent_runs_idempotency_active
    ON agent_runs(tenant_id, user_id, run_type, idempotency_key, created_at DESC)
    WHERE status IN ('queued', 'running', 'waiting_input');
