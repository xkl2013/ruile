-- Migration: 000001_tenant_provisioning_key
-- Description: Persist the idempotency key for enterprise workspace
-- provisioning. The key is nullable so ordinary tenants are unaffected.

ALTER TABLE tenants ADD COLUMN provisioning_key VARCHAR(128);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_provisioning_key
    ON tenants (provisioning_key);
