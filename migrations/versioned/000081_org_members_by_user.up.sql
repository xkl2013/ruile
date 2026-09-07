-- Migration: 000081_org_members_by_user
--
-- Shared-space membership is now granted to concrete user accounts. The table
-- name is kept for compatibility, but representative_user_id is the logical
-- member key; tenant_id is only the member-management source context.

DO $$ BEGIN RAISE NOTICE '[Migration 000081] switching shared-space membership key to account'; END $$;

DROP INDEX IF EXISTS idx_org_tenant_members_unique;

WITH ranked_members AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY organization_id, representative_user_id
               ORDER BY created_at ASC NULLS LAST, id ASC
           ) AS rn
      FROM organization_tenant_members
     WHERE representative_user_id <> ''
)
DELETE FROM organization_tenant_members otm
USING ranked_members ranked
WHERE otm.id = ranked.id
  AND ranked.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_tenant_members_unique
    ON organization_tenant_members (organization_id, representative_user_id)
    WHERE representative_user_id <> '';

CREATE INDEX IF NOT EXISTS idx_org_tenant_members_by_user
    ON organization_tenant_members (representative_user_id);

CREATE INDEX IF NOT EXISTS idx_org_tenant_members_by_tenant_user
    ON organization_tenant_members (tenant_id, representative_user_id);

COMMENT ON TABLE organization_tenant_members IS 'Shared-space members: one row grants one account access to an organization.';
COMMENT ON COLUMN organization_tenant_members.representative_user_id IS 'Account receiving the shared-space grant.';
