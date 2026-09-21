-- Generic AgentRun thread identity for multi-turn expert execution.

ALTER TABLE agent_runs
    ADD COLUMN IF NOT EXISTS thread_id VARCHAR(36) NOT NULL DEFAULT '';

UPDATE agent_runs
SET thread_id = id
WHERE thread_id = '';

CREATE INDEX IF NOT EXISTS idx_agent_runs_thread
    ON agent_runs(tenant_id, user_id, thread_id, created_at)
    WHERE deleted_at IS NULL;
