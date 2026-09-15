-- Migration: 000084_tenant_provisioning_key
-- Description: Persist the idempotency key for intent-based enterprise
-- workspace provisioning. NULL means the tenant was created outside that
-- command protocol.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS provisioning_key VARCHAR(128);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_provisioning_key
    ON tenants (provisioning_key);

COMMENT ON COLUMN tenants.provisioning_key IS
    'Hashed user-scoped idempotency key for enterprise workspace provisioning; NULL for ordinary tenants.';
