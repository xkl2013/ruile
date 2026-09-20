-- Migration: 000095_response_tier_config
-- Description: Store tenant-wide response tier bindings for all agents.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS response_tier_config JSONB DEFAULT NULL;

COMMENT ON COLUMN tenants.response_tier_config IS
    'Tenant-wide response tier bindings for all agents';
