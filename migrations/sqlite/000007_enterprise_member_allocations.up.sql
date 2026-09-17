-- Migration: 000007_enterprise_member_allocations
-- SQLite variant of enterprise member allocation tracking.

CREATE TABLE IF NOT EXISTS tenant_member_credit_allocations (
    id                                  TEXT PRIMARY KEY,
    tenant_id                           INTEGER NOT NULL REFERENCES tenants(id),
    user_id                             TEXT NOT NULL DEFAULT '',
    period_start_at                     DATETIME NOT NULL,
    period_end_at                       DATETIME NOT NULL,
    allocated_period_point_micros       INTEGER NOT NULL DEFAULT 0,
    allocated_balance_point_micros      INTEGER NOT NULL DEFAULT 0,
    status                              TEXT NOT NULL DEFAULT 'active',
    created_by_user_id                  TEXT NOT NULL DEFAULT '',
    updated_by_user_id                  TEXT NOT NULL DEFAULT '',
    snapshot_json                       JSON NOT NULL DEFAULT '{}',
    created_at                          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, user_id, period_start_at, period_end_at)
);

CREATE INDEX IF NOT EXISTS idx_tenant_member_credit_allocations_active
    ON tenant_member_credit_allocations(tenant_id, user_id, status, period_start_at, period_end_at);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_allocation
    ON tenant_usage_ledgers(tenant_id, allocation_id, usage_date DESC)
    WHERE allocation_id <> '';
