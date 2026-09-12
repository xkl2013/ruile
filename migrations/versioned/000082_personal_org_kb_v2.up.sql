-- Migration: 000082_personal_org_kb_v2
-- Description: Add compatibility metadata for personal/organization spaces,
-- shared-space scope, knowledge-base configuration provenance, and the
-- subscription reference table. No existing rows or access rules are changed.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS space_type VARCHAR(32);

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS sharing_scope VARCHAR(32);

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS config_source VARCHAR(32);

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS config_version VARCHAR(64);

CREATE TABLE IF NOT EXISTS knowledge_base_subscriptions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_kb_subscriptions_user_kb
    ON knowledge_base_subscriptions (user_id, knowledge_base_id);

CREATE INDEX IF NOT EXISTS idx_kb_subscriptions_user
    ON knowledge_base_subscriptions (user_id, status);

CREATE INDEX IF NOT EXISTS idx_kb_subscriptions_kb
    ON knowledge_base_subscriptions (knowledge_base_id, status);

COMMENT ON COLUMN tenants.space_type IS
    'Product workspace type: personal, organization, or legacy; NULL means unclassified.';

COMMENT ON COLUMN organizations.sharing_scope IS
    'Shared-space scope: legacy_cross_space or tenant_internal; NULL means unclassified.';

COMMENT ON COLUMN knowledge_bases.config_source IS
    'Configuration provenance: legacy or default.';

COMMENT ON COLUMN knowledge_bases.config_version IS
    'Resolved backend default configuration version at knowledge-base creation.';

COMMENT ON TABLE knowledge_base_subscriptions IS
    'User shortcuts to knowledge bases; subscription rows never grant access.';
