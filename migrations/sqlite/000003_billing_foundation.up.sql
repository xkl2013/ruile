-- Migration: 000003_billing_foundation
-- Additive SQLite equivalent of versioned migration 000086.

CREATE TABLE IF NOT EXISTS billing_plans (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    edition TEXT NOT NULL,
    space_type TEXT NOT NULL,
    status TEXT NOT NULL,
    is_public INTEGER NOT NULL DEFAULT 1,
    included_storage_bytes INTEGER NOT NULL DEFAULT 0,
    included_point_micros INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS billing_prices (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES billing_plans(id),
    code TEXT NOT NULL UNIQUE,
    currency TEXT NOT NULL,
    billing_interval TEXT NOT NULL,
    amount_minor INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0,
    effective_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_billing_prices_plan_id ON billing_prices(plan_id);

CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    plan_id TEXT NOT NULL REFERENCES billing_plans(id),
    status TEXT NOT NULL,
    billing_interval TEXT NOT NULL DEFAULT 'none',
    current_period_start DATETIME,
    current_period_end DATETIME,
    price_snapshot TEXT NOT NULL DEFAULT '{}',
    source TEXT NOT NULL DEFAULT 'migration',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant_id ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_plan_id ON tenant_subscriptions(plan_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_subscriptions_current
    ON tenant_subscriptions(tenant_id)
    WHERE status IN ('active', 'trialing', 'legacy');

CREATE TABLE IF NOT EXISTS tenant_credit_accounts (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id),
    balance_point_micros INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tenant_credit_transactions (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    account_id TEXT NOT NULL REFERENCES tenant_credit_accounts(id),
    type TEXT NOT NULL,
    amount_point_micros INTEGER NOT NULL,
    balance_point_micros INTEGER NOT NULL,
    ref_no TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_credit_transactions_tenant_id ON tenant_credit_transactions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_credit_transactions_account_id ON tenant_credit_transactions(account_id);

INSERT OR IGNORE INTO billing_plans (
    id, code, name, description, edition, space_type, status, is_public,
    included_storage_bytes, included_point_micros
) VALUES
    ('plan-personal-free-v1', 'personal_free', '个人免费版', '个人工作区默认套餐', 'personal', 'personal', 'active', 1, 1073741824, 0),
    ('plan-personal-pro-v1', 'personal_pro', '个人专业版', '个人工作区专业套餐', 'personal', 'personal', 'active', 1, 21474836480, 0),
    ('plan-enterprise-team-v1', 'enterprise_team', '企业团队版', '企业工作区默认套餐', 'enterprise', 'organization', 'active', 1, 107374182400, 0),
    ('plan-enterprise-business-v1', 'enterprise_business', '企业商务版', '企业工作区商务套餐', 'enterprise', 'organization', 'active', 1, 0, 0),
    ('plan-legacy-compat-v1', 'legacy_compat', '历史兼容版', '等待后台分类的历史工作区', 'legacy', 'legacy', 'active', 0, 0, 0);

INSERT OR IGNORE INTO billing_prices (
    id, plan_id, code, currency, billing_interval, amount_minor, status, is_default
) VALUES
    ('price-personal-free-cny-v1', 'plan-personal-free-v1', 'personal_free_cny_v1', 'CNY', 'none', 0, 'active', 1),
    ('price-personal-pro-cny-v1', 'plan-personal-pro-v1', 'personal_pro_cny_v1', 'CNY', 'month', 0, 'draft', 0),
    ('price-enterprise-team-cny-v1', 'plan-enterprise-team-v1', 'enterprise_team_cny_v1', 'CNY', 'month', 0, 'draft', 0),
    ('price-enterprise-business-cny-v1', 'plan-enterprise-business-v1', 'enterprise_business_cny_v1', 'CNY', 'month', 0, 'draft', 0),
    ('price-legacy-compat-cny-v1', 'plan-legacy-compat-v1', 'legacy_compat_cny_v1', 'CNY', 'none', 0, 'active', 1);

INSERT OR IGNORE INTO tenant_credit_accounts (id, tenant_id, balance_point_micros)
SELECT lower(hex(randomblob(16))), t.id, MAX(COALESCE(t.enterprise_credits, 0), 0) * 1000000
FROM tenants t
WHERE t.deleted_at IS NULL;

INSERT OR IGNORE INTO tenant_credit_transactions (
    id, tenant_id, account_id, type, amount_point_micros,
    balance_point_micros, ref_no, description
)
SELECT
    lower(hex(randomblob(16))),
    a.tenant_id,
    a.id,
    'legacy_import',
    a.balance_point_micros,
    a.balance_point_micros,
    'legacy-enterprise-credits:' || CAST(a.tenant_id AS TEXT),
    'Imported from tenants.enterprise_credits during billing foundation migration'
FROM tenant_credit_accounts a
WHERE a.balance_point_micros > 0;

INSERT OR IGNORE INTO tenant_subscriptions (
    id, tenant_id, plan_id, status, billing_interval, price_snapshot, source
)
SELECT
    lower(hex(randomblob(16))),
    t.id,
    CASE
        WHEN t.space_type = 'personal' THEN 'plan-personal-free-v1'
        WHEN t.space_type = 'organization' THEN 'plan-enterprise-team-v1'
        ELSE 'plan-legacy-compat-v1'
    END,
    CASE WHEN t.space_type IN ('personal', 'organization') THEN 'active' ELSE 'legacy' END,
    'none',
    json_object(
        'migration', '000003',
        'storage_quota_bytes', t.storage_quota,
        'legacy_enterprise_credits', COALESCE(t.enterprise_credits, 0)
    ),
    'migration'
FROM tenants t
WHERE t.deleted_at IS NULL;
