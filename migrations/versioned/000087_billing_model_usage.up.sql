-- Migration: 000087_billing_model_usage
-- Step 2A: versioned chat-model pricing, usage reservations, and immutable
-- tenant usage ledgers. Runtime enforcement remains default-off.

ALTER TABLE billing_plans
    ADD COLUMN IF NOT EXISTS billing_multiplier_ppm BIGINT NOT NULL DEFAULT 1000000;

CREATE TABLE IF NOT EXISTS billing_model_prices (
    id                                  VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    model_key                           VARCHAR(128) NOT NULL,
    provider                            VARCHAR(64) NOT NULL DEFAULT '',
    pricing_mode                        VARCHAR(32) NOT NULL DEFAULT 'token',
    input_nanousd_per_m_tokens          BIGINT NOT NULL DEFAULT 0,
    output_nanousd_per_m_tokens         BIGINT NOT NULL DEFAULT 0,
    cache_read_nanousd_per_m_tokens     BIGINT NOT NULL DEFAULT 0,
    cache_write_nanousd_per_m_tokens    BIGINT NOT NULL DEFAULT 0,
    call_nanousd_per_call               BIGINT NOT NULL DEFAULT 0,
    duration_nanousd_per_second         BIGINT NOT NULL DEFAULT 0,
    tiered_pricing_json                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    model_multiplier_ppm                BIGINT NOT NULL DEFAULT 1000000,
    version                             INTEGER NOT NULL DEFAULT 1,
    effective_at                        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at                          TIMESTAMP WITH TIME ZONE,
    status                              VARCHAR(32) NOT NULL DEFAULT 'active',
    snapshot_json                       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (model_key, version)
);

CREATE INDEX IF NOT EXISTS idx_billing_model_prices_active
    ON billing_model_prices(model_key, status, effective_at DESC);

CREATE TABLE IF NOT EXISTS tenant_usage_reservations (
    id                                  VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id                           BIGINT NOT NULL REFERENCES tenants(id),
    actor_user_id                       VARCHAR(64) NOT NULL DEFAULT '',
    usage_scope                         VARCHAR(32) NOT NULL DEFAULT 'personal_usage',
    allocation_id                       VARCHAR(36) NOT NULL DEFAULT '',
    ref_no                              VARCHAR(160) NOT NULL,
    model_key                           VARCHAR(128) NOT NULL DEFAULT '',
    pricing_id                          VARCHAR(36) NOT NULL DEFAULT '',
    estimated_base_cost_nanousd         BIGINT NOT NULL DEFAULT 0,
    estimated_billed_point_micros       BIGINT NOT NULL DEFAULT 0,
    reserved_period_point_micros        BIGINT NOT NULL DEFAULT 0,
    reserved_balance_point_micros       BIGINT NOT NULL DEFAULT 0,
    period_limit_point_micros           BIGINT NOT NULL DEFAULT 0,
    period_start_at                     TIMESTAMP WITH TIME ZONE,
    period_end_at                       TIMESTAMP WITH TIME ZONE,
    status                              VARCHAR(32) NOT NULL DEFAULT 'active',
    usage_ledger_id                     VARCHAR(36) NOT NULL DEFAULT '',
    expires_at                          TIMESTAMP WITH TIME ZONE NOT NULL,
    settled_at                          TIMESTAMP WITH TIME ZONE,
    released_at                         TIMESTAMP WITH TIME ZONE,
    reconciliation_at                   TIMESTAMP WITH TIME ZONE,
    failure_code                        VARCHAR(64) NOT NULL DEFAULT '',
    created_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_reservations_active
    ON tenant_usage_reservations(tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS tenant_usage_ledgers (
    id                                  VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id                           BIGINT NOT NULL REFERENCES tenants(id),
    actor_user_id                       VARCHAR(64) NOT NULL DEFAULT '',
    usage_scope                         VARCHAR(32) NOT NULL DEFAULT 'personal_usage',
    allocation_id                       VARCHAR(36) NOT NULL DEFAULT '',
    ref_no                              VARCHAR(160) NOT NULL,
    source                              VARCHAR(32) NOT NULL DEFAULT 'web',
    session_id                          VARCHAR(64) NOT NULL DEFAULT '',
    service_code                        VARCHAR(64) NOT NULL DEFAULT 'chat.completion',
    model_id                            VARCHAR(64) NOT NULL DEFAULT '',
    model_key                           VARCHAR(128) NOT NULL DEFAULT '',
    provider                            VARCHAR(64) NOT NULL DEFAULT '',
    pricing_id                          VARCHAR(36) NOT NULL DEFAULT '',
    pricing_version                     INTEGER NOT NULL DEFAULT 0,
    input_tokens                        BIGINT NOT NULL DEFAULT 0,
    cached_tokens                       BIGINT NOT NULL DEFAULT 0,
    output_tokens                       BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens                    BIGINT NOT NULL DEFAULT 0,
    call_count                          BIGINT NOT NULL DEFAULT 1,
    duration_millis                     BIGINT NOT NULL DEFAULT 0,
    base_cost_nanousd                   BIGINT NOT NULL DEFAULT 0,
    rated_cost_nanousd                  BIGINT NOT NULL DEFAULT 0,
    billed_point_micros                 BIGINT NOT NULL DEFAULT 0,
    period_covered_point_micros         BIGINT NOT NULL DEFAULT 0,
    balance_charged_point_micros        BIGINT NOT NULL DEFAULT 0,
    balance_after_point_micros          BIGINT,
    status                              VARCHAR(32) NOT NULL DEFAULT 'observed',
    failure_code                        VARCHAR(64) NOT NULL DEFAULT '',
    billing_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    usage_date                          DATE NOT NULL DEFAULT CURRENT_DATE,
    pricing_snapshot_json               JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_tenant_date
    ON tenant_usage_ledgers(tenant_id, usage_date DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_ledgers_actor_date
    ON tenant_usage_ledgers(tenant_id, actor_user_id, usage_date DESC);

