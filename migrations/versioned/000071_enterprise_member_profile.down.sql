-- Reverse migration for 000071_enterprise_member_profile.

DO $$ BEGIN RAISE NOTICE '[Migration 000071 down] Reverting tenant_members enterprise profile columns'; END $$;

DROP INDEX IF EXISTS idx_tenant_members_tenant_department;
DROP INDEX IF EXISTS idx_tenant_members_tenant_source;
DROP INDEX IF EXISTS idx_tenant_members_tenant_status;

ALTER TABLE tenant_members
    DROP COLUMN IF EXISTS suspended_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS department,
    DROP COLUMN IF EXISTS external_user_id,
    DROP COLUMN IF EXISTS source;

DO $$ BEGIN RAISE NOTICE '[Migration 000071 down] tenant_members enterprise profile columns reverted'; END $$;
