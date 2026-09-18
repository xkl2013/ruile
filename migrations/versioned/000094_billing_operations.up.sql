-- Migration: 000094_billing_operations
-- Step 3A: deployment-wide purchase catalog plus durable offline commercial
-- operations. Online providers remain disabled; SystemAdmin manual orders are
-- recorded as paid with provider=manual and issue entitlements atomically.

CREATE TABLE IF NOT EXISTS billing_purchase_items (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    item_type VARCHAR(32) NOT NULL
        CHECK (item_type IN ('topup', 'storage_addon')),
    edition_scope VARCHAR(32) NOT NULL DEFAULT 'all'
        CHECK (edition_scope IN ('all', 'personal', 'enterprise')),
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
    amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
    credit_point_micros BIGINT NOT NULL DEFAULT 0,
    storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    duration_days INTEGER NOT NULL DEFAULT 0 CHECK (duration_days >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    sort_order INTEGER NOT NULL DEFAULT 0,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_billing_purchase_item_entitlement CHECK (
        (item_type = 'topup' AND credit_point_micros > 0 AND storage_quota_bytes = 0)
        OR
        (item_type = 'storage_addon' AND storage_quota_bytes > 0 AND credit_point_micros = 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_billing_purchase_items_type_status
    ON billing_purchase_items (item_type, edition_scope, status, sort_order);

CREATE TABLE IF NOT EXISTS billing_payment_orders (
    id VARCHAR(36) PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    order_type VARCHAR(32) NOT NULL
        CHECK (order_type IN ('subscription', 'topup', 'storage_addon', 'manual_contract')),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
    plan_id VARCHAR(36) NOT NULL DEFAULT '',
    price_id VARCHAR(36) NOT NULL DEFAULT '',
    item_id VARCHAR(36) NOT NULL DEFAULT '',
    provider VARCHAR(32) NOT NULL DEFAULT '',
    payment_method VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'paid', 'failed', 'expired', 'closed', 'reconciliation')),
    currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
    amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
    credit_point_micros BIGINT NOT NULL DEFAULT 0,
    storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    billing_interval VARCHAR(16) NOT NULL DEFAULT '',
    cycles INTEGER NOT NULL DEFAULT 1 CHECK (cycles > 0),
    external_payment_ref VARCHAR(128) NOT NULL DEFAULT '',
    external_checkout_ref VARCHAR(128) NOT NULL DEFAULT '',
    checkout_url TEXT NOT NULL DEFAULT '',
    qr_code_url TEXT NOT NULL DEFAULT '',
    notify_url TEXT NOT NULL DEFAULT '',
    return_url TEXT NOT NULL DEFAULT '',
    provider_payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    notify_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    paid_at TIMESTAMP WITH TIME ZONE,
    expired_at TIMESTAMP WITH TIME ZONE,
    snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_billing_payment_orders_tenant_created
    ON billing_payment_orders (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_billing_payment_orders_status_expired
    ON billing_payment_orders (status, expired_at);

CREATE TABLE IF NOT EXISTS tenant_storage_addon_grants (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    item_id VARCHAR(36) NOT NULL DEFAULT '',
    storage_quota_bytes BIGINT NOT NULL CHECK (storage_quota_bytes > 0),
    status VARCHAR(32) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'expired', 'revoked')),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_storage_addon_grants_tenant
    ON tenant_storage_addon_grants (tenant_id, status, expires_at);

-- Initial catalog values are only used by the SystemAdmin operations console;
-- the client has no online checkout while payment providers are disabled.
INSERT INTO billing_purchase_items (
    id, code, item_type, edition_scope, name, description, currency,
    amount_cents, credit_point_micros, storage_quota_bytes, duration_days,
    status, sort_order, metadata_json
) VALUES
    ('purchase-credit-1000-v1', 'credit_1000_points', 'topup', 'all', '1,000 积分', '适用于个人和企业工作区的积分包', 'CNY', 990, 1000000000, 0, 0, 'active', 10, '{"bootstrap":true}'::jsonb),
    ('purchase-credit-10000-v1', 'credit_10000_points', 'topup', 'all', '10,000 积分', '适用于个人和企业工作区的积分包', 'CNY', 9900, 10000000000, 0, 0, 'active', 20, '{"bootstrap":true}'::jsonb),
    ('purchase-storage-20gb-v1', 'storage_20gb', 'storage_addon', 'all', '20 GB 存储包', '适用于个人和企业工作区的长期存储扩容', 'CNY', 2900, 0, 21474836480, 0, 'active', 30, '{"bootstrap":true}'::jsonb),
    ('purchase-storage-100gb-v1', 'storage_100gb', 'storage_addon', 'enterprise', '100 GB 存储包', '适用于企业工作区的长期存储扩容', 'CNY', 9900, 0, 107374182400, 0, 'active', 40, '{"bootstrap":true}'::jsonb)
ON CONFLICT (code) DO NOTHING;
