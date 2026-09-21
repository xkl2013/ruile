DELETE FROM agent_runs WHERE run_type = 'expert_follow_up';

CREATE TABLE agent_runs_old (
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
    CHECK (run_type IN ('service_daily_report', 'service_memory_extract', 'expert_agent_test'))
);

INSERT INTO agent_runs_old
SELECT * FROM agent_runs;

DROP TABLE agent_runs;
ALTER TABLE agent_runs_old RENAME TO agent_runs;

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
