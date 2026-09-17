-- Migration: 000004_billing_model_usage
-- SQLite equivalent of versioned migration 000087.

ALTER TABLE billing_plans
    ADD COLUMN billing_multiplier_ppm INTEGER NOT NULL DEFAULT 1000000;

CREATE TABLE IF NOT EXISTS billing_model_prices (
    id TEXT PRIMARY KEY,
    model_key TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    pricing_mode TEXT NOT NULL DEFAULT 'token',
    input_nanousd_per_m_tokens INTEGER NOT NULL DEFAULT 0,
    output_nanousd_per_m_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_nanousd_per_m_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_nanousd_per_m_tokens INTEGER NOT NULL DEFAULT 0,
    call_nanousd_per_call INTEGER NOT NULL DEFAULT 0,
    duration_nanousd_per_second INTEGER NOT NULL DEFAULT 0,
    tiered_pricing_json TEXT NOT NULL DEFAULT '{}',
    model_multiplier_ppm INTEGER NOT NULL DEFAULT 1000000,
    version INTEGER NOT NULL DEFAULT 1,
    effective_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    status TEXT NOT NULL DEFAULT 'active',
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (model_key, version)
);
CREATE INDEX IF NOT EXISTS idx_billing_model_prices_active
    ON billing_model_prices(model_key, status, effective_at DESC);

CREATE TABLE IF NOT EXISTS tenant_usage_reservations (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    actor_user_id TEXT NOT NULL DEFAULT '',
    usage_scope TEXT NOT NULL DEFAULT 'personal_usage',
    allocation_id TEXT NOT NULL DEFAULT '',
    ref_no TEXT NOT NULL,
    model_key TEXT NOT NULL DEFAULT '',
    pricing_id TEXT NOT NULL DEFAULT '',
    estimated_base_cost_nanousd INTEGER NOT NULL DEFAULT 0,
    estimated_billed_point_micros INTEGER NOT NULL DEFAULT 0,
    reserved_period_point_micros INTEGER NOT NULL DEFAULT 0,
    reserved_balance_point_micros INTEGER NOT NULL DEFAULT 0,
    period_limit_point_micros INTEGER NOT NULL DEFAULT 0,
    period_start_at DATETIME,
    period_end_at DATETIME,
    status TEXT NOT NULL DEFAULT 'active',
    usage_ledger_id TEXT NOT NULL DEFAULT '',
    expires_at DATETIME NOT NULL,
    settled_at DATETIME,
    released_at DATETIME,
    reconciliation_at DATETIME,
    failure_code TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_reservations_active
    ON tenant_usage_reservations(tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS tenant_usage_ledgers (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    actor_user_id TEXT NOT NULL DEFAULT '',
    usage_scope TEXT NOT NULL DEFAULT 'personal_usage',
    allocation_id TEXT NOT NULL DEFAULT '',
    ref_no TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'web',
    session_id TEXT NOT NULL DEFAULT '',
    service_code TEXT NOT NULL DEFAULT 'chat.completion',
    model_id TEXT NOT NULL DEFAULT '',
    model_key TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    pricing_id TEXT NOT NULL DEFAULT '',
    pricing_version INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    call_count INTEGER NOT NULL DEFAULT 1,
    duration_millis INTEGER NOT NULL DEFAULT 0,
    base_cost_nanousd INTEGER NOT NULL DEFAULT 0,
    rated_cost_nanousd INTEGER NOT NULL DEFAULT 0,
    billed_point_micros INTEGER NOT NULL DEFAULT 0,
    period_covered_point_micros INTEGER NOT NULL DEFAULT 0,
    balance_charged_point_micros INTEGER NOT NULL DEFAULT 0,
    balance_after_point_micros INTEGER,
    status TEXT NOT NULL DEFAULT 'observed',
    failure_code TEXT NOT NULL DEFAULT '',
    billing_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    usage_date DATE NOT NULL DEFAULT CURRENT_DATE,
    pricing_snapshot_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_tenant_date
    ON tenant_usage_ledgers(tenant_id, usage_date DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_actor_date
    ON tenant_usage_ledgers(tenant_id, actor_user_id, usage_date DESC);

