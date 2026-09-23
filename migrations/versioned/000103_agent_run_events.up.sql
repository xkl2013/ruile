-- Migration 000103: Durable, ordered events for WorkBuddy-style AgentRun playback.

CREATE TABLE IF NOT EXISTS agent_run_event_sequences (
    run_id VARCHAR(36) PRIMARY KEY REFERENCES agent_runs(id) ON DELETE CASCADE,
    next_sequence BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS agent_run_events (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    run_id VARCHAR(36) NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    sequence BIGINT NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_run_events_sequence
    ON agent_run_events(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_replay
    ON agent_run_events(run_id, sequence, created_at);
