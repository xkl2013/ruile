DROP INDEX IF EXISTS idx_agent_runs_thread;

ALTER TABLE agent_runs
    DROP COLUMN thread_id;
