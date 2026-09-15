DROP INDEX IF EXISTS uq_tenants_provisioning_key;

ALTER TABLE tenants DROP COLUMN provisioning_key;
