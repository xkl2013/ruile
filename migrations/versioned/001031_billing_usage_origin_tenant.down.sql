DROP INDEX IF EXISTS idx_tenant_usage_ledgers_origin_tenant_date;
DROP INDEX IF EXISTS idx_tenant_usage_reservations_origin_tenant;

ALTER TABLE tenant_usage_ledgers DROP COLUMN IF EXISTS origin_tenant_id;
ALTER TABLE tenant_usage_reservations DROP COLUMN IF EXISTS origin_tenant_id;
