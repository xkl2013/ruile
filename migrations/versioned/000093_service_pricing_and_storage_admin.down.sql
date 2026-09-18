ALTER TABLE tenant_usage_ledgers
    DROP COLUMN IF EXISTS service_units,
    DROP COLUMN IF EXISTS service_pricing_version,
    DROP COLUMN IF EXISTS service_pricing_id;

ALTER TABLE tenant_usage_reservations
    DROP COLUMN IF EXISTS service_pricing_id;

DROP TABLE IF EXISTS billing_service_prices;
