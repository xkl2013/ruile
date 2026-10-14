DROP INDEX IF EXISTS idx_tenant_usage_ledgers_origin_tenant_date;
DROP INDEX IF EXISTS idx_tenant_usage_reservations_origin_tenant;
-- SQLite keeps additive attribution columns on rollback for compatibility.
