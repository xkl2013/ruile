DROP TABLE IF EXISTS tenant_usage_ledgers;
DROP TABLE IF EXISTS tenant_usage_reservations;
DROP TABLE IF EXISTS billing_model_prices;
ALTER TABLE billing_plans DROP COLUMN billing_multiplier_ppm;

