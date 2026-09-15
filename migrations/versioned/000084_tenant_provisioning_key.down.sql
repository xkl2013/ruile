DROP INDEX IF EXISTS uq_tenants_provisioning_key;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS provisioning_key;
