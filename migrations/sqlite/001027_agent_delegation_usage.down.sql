DROP INDEX IF EXISTS idx_tenant_usage_ledgers_delegation_run;
DROP INDEX IF EXISTS idx_tenant_usage_reservations_delegation_run;
DROP TABLE IF EXISTS agent_delegation_runs;
-- SQLite keeps additive attribution columns on rollback for compatibility.
