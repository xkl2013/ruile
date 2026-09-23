-- Service-space experts replaced the legacy employee-profile binding model.
DROP INDEX IF EXISTS idx_agent_bindings_unique;
DROP TABLE IF EXISTS agent_bindings;
