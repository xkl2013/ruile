-- Migration 000101: Expert AgentRun types.

ALTER TABLE agent_runs
    DROP CONSTRAINT IF EXISTS chk_agent_runs_type;

ALTER TABLE agent_runs
    ADD CONSTRAINT chk_agent_runs_type
    CHECK (run_type IN ('service_daily_report', 'service_memory_extract', 'expert_agent_test', 'expert_follow_up'));
