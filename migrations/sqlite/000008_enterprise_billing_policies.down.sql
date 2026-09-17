DROP INDEX IF EXISTS idx_tenant_member_credit_allocations_policy;

ALTER TABLE tenant_member_credit_allocations DROP COLUMN overage_policy;
ALTER TABLE tenant_member_credit_allocations DROP COLUMN monthly_limit_point_micros;
ALTER TABLE tenant_member_credit_allocations DROP COLUMN limit_mode;

DROP TABLE IF EXISTS tenant_billing_policies;
