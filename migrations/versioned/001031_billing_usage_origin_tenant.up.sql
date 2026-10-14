-- Preserve the workspace where usage originated when another workspace pays.

ALTER TABLE tenant_usage_reservations
    ADD COLUMN IF NOT EXISTS origin_tenant_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tenant_usage_ledgers
    ADD COLUMN IF NOT EXISTS origin_tenant_id BIGINT NOT NULL DEFAULT 0;

UPDATE tenant_usage_reservations
SET origin_tenant_id = tenant_id
WHERE origin_tenant_id = 0;

UPDATE tenant_usage_ledgers
SET origin_tenant_id = tenant_id
WHERE origin_tenant_id = 0;

CREATE INDEX IF NOT EXISTS idx_tenant_usage_reservations_origin_tenant
    ON tenant_usage_reservations(origin_tenant_id, status, expires_at);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_origin_tenant_date
    ON tenant_usage_ledgers(origin_tenant_id, usage_date DESC, created_at DESC);
