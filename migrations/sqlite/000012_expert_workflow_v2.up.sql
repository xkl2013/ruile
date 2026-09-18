-- Expert workflow V2. SQLite requires rebuilding agent_runs to update CHECK constraints.

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
    CHECK (status IN ('queued', 'running', 'waiting_input', 'succeeded', 'failed', 'cancelled')),
    CHECK (run_type IN ('service_daily_report', 'service_memory_extract', 'expert_agent_test'))
);

INSERT INTO agent_runs_new (
    id, tenant_id, user_id, profile_id, run_type, agent_ref, agent_version,
    trigger_type, trigger_id, status, input, result, error_code, error_message,
    idempotency_key, task_id, attempt, queued_at, started_at, finished_at,
    created_at, updated_at, deleted_at
)
SELECT
    id, tenant_id, user_id, profile_id, run_type, agent_ref, agent_version,
    trigger_type, trigger_id, status, input, result, error_code, error_message,
    idempotency_key, task_id, attempt, queued_at, started_at, finished_at,
    created_at, updated_at, deleted_at
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
CREATE UNIQUE INDEX idx_agent_runs_idempotency_active
    ON agent_runs(tenant_id, user_id, run_type, idempotency_key, created_at DESC)
    WHERE status IN ('queued', 'running', 'waiting_input');

CREATE TABLE agent_run_steps (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES agent_runs(id),
    sequence INTEGER NOT NULL,
    step_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    model_id TEXT NOT NULL DEFAULT '',
    input TEXT NOT NULL DEFAULT '{}',
    output TEXT NOT NULL DEFAULT '{}',
    error TEXT NOT NULL DEFAULT '',
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (status IN ('running', 'succeeded', 'failed'))
);
CREATE UNIQUE INDEX idx_agent_run_steps_sequence
    ON agent_run_steps(run_id, sequence);

CREATE TABLE agent_run_input_revisions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES agent_runs(id),
    revision INTEGER NOT NULL,
    source TEXT NOT NULL DEFAULT 'request',
    input TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX idx_agent_run_input_revisions_revision
    ON agent_run_input_revisions(run_id, revision);

CREATE TABLE agent_requirement_snapshots (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES agent_runs(id),
    revision INTEGER NOT NULL,
    snapshot_values TEXT NOT NULL DEFAULT '{}',
    assumptions TEXT NOT NULL DEFAULT '[]',
    missing TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX idx_agent_requirement_snapshots_revision
    ON agent_requirement_snapshots(run_id, revision);
