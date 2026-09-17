DROP INDEX IF EXISTS idx_tenant_member_credit_allocations_policy;

ALTER TABLE tenant_member_credit_allocations
    DROP COLUMN IF EXISTS overage_policy,
    DROP COLUMN IF EXISTS monthly_limit_point_micros,
    DROP COLUMN IF EXISTS limit_mode;

DROP TABLE IF EXISTS tenant_billing_policies;
