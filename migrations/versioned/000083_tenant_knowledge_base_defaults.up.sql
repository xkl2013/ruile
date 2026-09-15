-- Migration: 000083_tenant_knowledge_base_defaults
-- Stores the advanced knowledge-base configuration at workspace scope.
-- The API applies this configuration to every non-temporary KB in the
-- workspace and inherits it for future KB creation.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS knowledge_base_defaults_config JSONB;

COMMENT ON COLUMN tenants.knowledge_base_defaults_config IS
    'Advanced knowledge-base configuration shared by all non-temporary knowledge bases in this workspace.';
