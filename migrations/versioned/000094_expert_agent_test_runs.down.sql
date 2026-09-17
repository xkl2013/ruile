-- Migration: 000094_expert_agent_test_runs (rollback)

DELETE FROM agent_runs WHERE run_type = 'expert_agent_test';

ALTER TABLE agent_runs
    DROP CONSTRAINT IF EXISTS chk_agent_runs_type;

ALTER TABLE agent_runs
    ADD CONSTRAINT chk_agent_runs_type
    CHECK (run_type IN ('service_daily_report', 'service_memory_extract'));
