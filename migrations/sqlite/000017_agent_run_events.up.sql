-- Durable, ordered events for WorkBuddy-style AgentRun playback.

CREATE TABLE IF NOT EXISTS agent_run_event_sequences (
    run_id TEXT PRIMARY KEY REFERENCES agent_runs(id) ON DELETE CASCADE,
    next_sequence INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS agent_run_events (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_run_events_sequence
    ON agent_run_events(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_replay
    ON agent_run_events(run_id, sequence, created_at);
