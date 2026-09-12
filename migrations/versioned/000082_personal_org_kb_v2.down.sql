-- Reverse migration for 000082_personal_org_kb_v2.

DROP TABLE IF EXISTS knowledge_base_subscriptions;

ALTER TABLE knowledge_bases
    DROP COLUMN IF EXISTS config_version;

ALTER TABLE knowledge_bases
    DROP COLUMN IF EXISTS config_source;

ALTER TABLE organizations
    DROP COLUMN IF EXISTS sharing_scope;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS space_type;
