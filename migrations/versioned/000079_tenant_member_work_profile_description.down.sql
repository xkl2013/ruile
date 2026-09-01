-- Reverse migration for 000079_tenant_member_work_profile_description.

DO $$ BEGIN RAISE NOTICE '[Migration 000079 down] Dropping tenant member work profile description'; END $$;

ALTER TABLE tenant_members
    DROP COLUMN IF EXISTS work_profile_description;

DO $$ BEGIN RAISE NOTICE '[Migration 000079 down] tenant member work profile description reverted'; END $$;
