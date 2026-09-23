DROP INDEX IF EXISTS idx_agent_runs_service_thread;
DROP INDEX IF EXISTS idx_agent_runs_service;
ALTER TABLE agent_runs DROP COLUMN IF EXISTS service_id;

DROP INDEX IF EXISTS idx_sessions_service;
ALTER TABLE sessions DROP COLUMN IF EXISTS expert_name;
ALTER TABLE sessions DROP COLUMN IF EXISTS expert_ref;
ALTER TABLE sessions DROP COLUMN IF EXISTS service_id;

DROP TABLE IF EXISTS service_artifacts;
DROP TABLE IF EXISTS service_expert_bindings;
DROP TABLE IF EXISTS service_members;
DROP TABLE IF EXISTS services;
