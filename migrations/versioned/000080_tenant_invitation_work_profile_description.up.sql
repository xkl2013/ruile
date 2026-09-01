-- Migration: 000080_tenant_invitation_work_profile_description
-- Stores the operator-authored member avatar / work profile description on
-- targeted invitations so accepting the invitation can create a complete
-- tenant_members row.

DO $$ BEGIN RAISE NOTICE '[Migration 000080] Adding work_profile_description to tenant_invitations'; END $$;

ALTER TABLE tenant_invitations
    ADD COLUMN IF NOT EXISTS work_profile_description TEXT NOT NULL DEFAULT '';

DO $$ BEGIN RAISE NOTICE '[Migration 000080] tenant_invitations work_profile_description ready'; END $$;
