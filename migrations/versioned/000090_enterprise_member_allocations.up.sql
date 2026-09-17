-- Migration: 000090_enterprise_member_allocations
-- Step 2B partial: enterprise member allocation envelope and user-level usage
-- attribution. Existing usage ledgers remain valid when allocation_id is empty.

CREATE TABLE IF NOT EXISTS tenant_member_credit_allocations (
    id                                  VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id                           BIGINT NOT NULL REFERENCES tenants(id),
    user_id                             VARCHAR(64) NOT NULL DEFAULT '',
    period_start_at                     TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end_at                       TIMESTAMP WITH TIME ZONE NOT NULL,
    allocated_period_point_micros       BIGINT NOT NULL DEFAULT 0,
    allocated_balance_point_micros      BIGINT NOT NULL DEFAULT 0,
    status                              VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id                  VARCHAR(64) NOT NULL DEFAULT '',
    updated_by_user_id                  VARCHAR(64) NOT NULL DEFAULT '',
    snapshot_json                       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_member_credit_allocations_period
    ON tenant_member_credit_allocations(tenant_id, user_id, period_start_at, period_end_at);

CREATE INDEX IF NOT EXISTS idx_tenant_member_credit_allocations_active
    ON tenant_member_credit_allocations(tenant_id, user_id, status, period_start_at, period_end_at);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_allocation
    ON tenant_usage_ledgers(tenant_id, allocation_id, usage_date DESC)
    WHERE allocation_id <> '';
