-- Reverse migration for 000080_tenant_invitation_work_profile_description.

DO $$ BEGIN RAISE NOTICE '[Migration 000080] Dropping work_profile_description from tenant_invitations'; END $$;

ALTER TABLE tenant_invitations
    DROP COLUMN IF EXISTS work_profile_description;

DO $$ BEGIN RAISE NOTICE '[Migration 000080] tenant_invitations work_profile_description dropped'; END $$;
