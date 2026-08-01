-- Migration: 000071_enterprise_member_profile
-- Adds enterprise directory/lifecycle metadata to tenant_members. These fields
-- are deliberately optional so existing self-hosted/community deployments keep
-- working, while enterprise deployments can sync from OIDC/SCIM/LDAP/HRIS jobs.

DO $$ BEGIN RAISE NOTICE '[Migration 000071] Extending tenant_members enterprise profile columns'; END $$;

ALTER TABLE tenant_members
    ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS external_user_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS department VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_status
    ON tenant_members(tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_source
    ON tenant_members(tenant_id, source)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_department
    ON tenant_members(tenant_id, department)
    WHERE deleted_at IS NULL;

DO $$ BEGIN RAISE NOTICE '[Migration 000071] tenant_members enterprise profile columns ready'; END $$;
