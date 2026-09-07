-- Reverse migration for 000081_org_members_by_user.
--
-- Restoring the old tenant-level unique key is only possible after duplicate
-- account rows in the same (organization_id, tenant_id) pair are removed.
DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
      FROM (
        SELECT organization_id, tenant_id
          FROM organization_tenant_members
         GROUP BY organization_id, tenant_id
        HAVING COUNT(*) > 1
      ) d;

    IF dup_count > 0 THEN
        RAISE EXCEPTION
            '[Migration 000081 down] cannot restore tenant-level unique index while duplicate tenant rows exist: % duplicate group(s)',
            dup_count;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_org_tenant_members_unique;
DROP INDEX IF EXISTS idx_org_tenant_members_by_user;
DROP INDEX IF EXISTS idx_org_tenant_members_by_tenant_user;

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_tenant_members_unique
    ON organization_tenant_members (organization_id, tenant_id);

COMMENT ON TABLE organization_tenant_members IS 'Shared-space members grouped by tenant.';
COMMENT ON COLUMN organization_tenant_members.representative_user_id IS 'Display user attached to the tenant membership row.';
