DROP INDEX IF EXISTS idx_tenant_usage_ledgers_delegation_run;
DROP INDEX IF EXISTS idx_tenant_usage_reservations_delegation_run;

ALTER TABLE tenant_usage_ledgers DROP COLUMN IF EXISTS agent_id;
ALTER TABLE tenant_usage_ledgers DROP COLUMN IF EXISTS delegation_run_id;
ALTER TABLE tenant_usage_reservations DROP COLUMN IF EXISTS agent_id;
ALTER TABLE tenant_usage_reservations DROP COLUMN IF EXISTS delegation_run_id;

DROP TABLE IF EXISTS agent_delegation_runs;
