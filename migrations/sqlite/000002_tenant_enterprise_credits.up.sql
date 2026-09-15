-- Migration: 000002_tenant_enterprise_credits
-- Description: Store enterprise credits assigned by the temporary
-- SystemAdmin manual enterprise provisioning flow.

ALTER TABLE tenants ADD COLUMN enterprise_credits BIGINT NOT NULL DEFAULT 0;
