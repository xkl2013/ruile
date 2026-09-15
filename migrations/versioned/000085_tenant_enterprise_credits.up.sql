-- Migration: 000085_tenant_enterprise_credits
-- Description: Store enterprise credits assigned by the temporary
-- SystemAdmin manual enterprise provisioning flow.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS enterprise_credits BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN tenants.enterprise_credits IS
    'Enterprise credit balance assigned by SystemAdmin manual provisioning.';
