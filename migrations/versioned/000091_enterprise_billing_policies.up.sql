-- Migration: 000091_enterprise_billing_policies
-- Enterprise resource-pool policy and member monthly usage limits.

CREATE TABLE IF NOT EXISTS tenant_billing_policies (
    tenant_id                                           BIGINT PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    default_member_monthly_limit_point_micros           BIGINT NOT NULL DEFAULT 100000000,
    member_overage_policy                               VARCHAR(32) NOT NULL DEFAULT 'block',
    updated_by_user_id                                  VARCHAR(64) NOT NULL DEFAULT '',
    created_at                                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_tenant_billing_policy_limit
        CHECK (default_member_monthly_limit_point_micros >= 0),
    CONSTRAINT chk_tenant_billing_policy_overage
        CHECK (member_overage_policy IN ('block', 'use_enterprise_balance'))
);

ALTER TABLE tenant_member_credit_allocations
    ADD COLUMN IF NOT EXISTS limit_mode VARCHAR(32) NOT NULL DEFAULT 'inherit',
    ADD COLUMN IF NOT EXISTS monthly_limit_point_micros BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS overage_policy VARCHAR(32) NOT NULL DEFAULT 'inherit';

UPDATE tenant_member_credit_allocations
SET
    monthly_limit_point_micros =
        allocated_period_point_micros + allocated_balance_point_micros,
    limit_mode = CASE
        WHEN allocated_period_point_micros + allocated_balance_point_micros > 0
            THEN 'custom'
        ELSE 'inherit'
    END,
    overage_policy = 'inherit'
WHERE
    monthly_limit_point_micros = 0
    AND limit_mode = 'inherit';

INSERT INTO tenant_billing_policies (
    tenant_id,
    default_member_monthly_limit_point_micros,
    member_overage_policy
)
SELECT
    id,
    100000000,
    'block'
FROM tenants
WHERE space_type = 'organization' AND deleted_at IS NULL
ON CONFLICT (tenant_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_tenant_member_credit_allocations_policy
    ON tenant_member_credit_allocations(tenant_id, user_id, limit_mode, status);
