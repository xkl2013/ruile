-- Migration: 000008_enterprise_billing_policies
-- SQLite variant of enterprise resource-pool policy and member monthly limits.

CREATE TABLE IF NOT EXISTS tenant_billing_policies (
    tenant_id                                           INTEGER PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    default_member_monthly_limit_point_micros           INTEGER NOT NULL DEFAULT 100000000 CHECK (default_member_monthly_limit_point_micros >= 0),
    member_overage_policy                               TEXT NOT NULL DEFAULT 'block' CHECK (member_overage_policy IN ('block', 'use_enterprise_balance')),
    updated_by_user_id                                  TEXT NOT NULL DEFAULT '',
    created_at                                          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                                          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE tenant_member_credit_allocations
    ADD COLUMN limit_mode TEXT NOT NULL DEFAULT 'inherit';

ALTER TABLE tenant_member_credit_allocations
    ADD COLUMN monthly_limit_point_micros INTEGER NOT NULL DEFAULT 0;

ALTER TABLE tenant_member_credit_allocations
    ADD COLUMN overage_policy TEXT NOT NULL DEFAULT 'inherit';

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

INSERT OR IGNORE INTO tenant_billing_policies (
    tenant_id,
    default_member_monthly_limit_point_micros,
    member_overage_policy
)
SELECT
    id,
    100000000,
    'block'
FROM tenants
WHERE space_type = 'organization' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_member_credit_allocations_policy
    ON tenant_member_credit_allocations(tenant_id, user_id, limit_mode, status);
