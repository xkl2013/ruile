-- Migration: 000079_tenant_member_work_profile_description
-- Stores the member-level work avatar description used as service-routing input.

DO $$ BEGIN RAISE NOTICE '[Migration 000079] Adding tenant member work profile description'; END $$;

ALTER TABLE tenant_members
    ADD COLUMN IF NOT EXISTS work_profile_description TEXT NOT NULL DEFAULT '';

DO $$ BEGIN RAISE NOTICE '[Migration 000079] tenant member work profile description ready'; END $$;
