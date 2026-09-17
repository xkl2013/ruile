-- Migration: 000086_billing_foundation
-- Step 1 billing foundation. This migration is additive and does not change
-- existing tenant quotas, ownership, knowledge-base placement, or file paths.

CREATE TABLE IF NOT EXISTS billing_plans (
    id                       VARCHAR(36) PRIMARY KEY,
    code                     VARCHAR(64) NOT NULL UNIQUE,
    name                     VARCHAR(128) NOT NULL,
    description              TEXT NOT NULL DEFAULT '',
    edition                  VARCHAR(32) NOT NULL,
    space_type               VARCHAR(32) NOT NULL,
    status                   VARCHAR(32) NOT NULL,
    is_public                BOOLEAN NOT NULL DEFAULT TRUE,
    included_storage_bytes   BIGINT NOT NULL DEFAULT 0,
    included_point_micros    BIGINT NOT NULL DEFAULT 0,
    created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS billing_prices (
    id                VARCHAR(36) PRIMARY KEY,
    plan_id           VARCHAR(36) NOT NULL REFERENCES billing_plans(id),
    code              VARCHAR(64) NOT NULL UNIQUE,
    currency          VARCHAR(8) NOT NULL,
    billing_interval  VARCHAR(16) NOT NULL,
    amount_minor      BIGINT NOT NULL DEFAULT 0,
    status            VARCHAR(32) NOT NULL,
    is_default        BOOLEAN NOT NULL DEFAULT FALSE,
    effective_at      TIMESTAMP WITH TIME ZONE,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_billing_prices_plan_id ON billing_prices(plan_id);

CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id                    VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id             BIGINT NOT NULL REFERENCES tenants(id),
    plan_id               VARCHAR(36) NOT NULL REFERENCES billing_plans(id),
    status                VARCHAR(32) NOT NULL,
    billing_interval      VARCHAR(16) NOT NULL DEFAULT 'none',
    current_period_start  TIMESTAMP WITH TIME ZONE,
    current_period_end    TIMESTAMP WITH TIME ZONE,
    price_snapshot        JSONB NOT NULL DEFAULT '{}'::jsonb,
    source                VARCHAR(32) NOT NULL DEFAULT 'migration',
    created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant_id ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_plan_id ON tenant_subscriptions(plan_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_subscriptions_current
    ON tenant_subscriptions(tenant_id)
    WHERE status IN ('active', 'trialing', 'legacy');

CREATE TABLE IF NOT EXISTS tenant_credit_accounts (
    id                    VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id             BIGINT NOT NULL UNIQUE REFERENCES tenants(id),
    balance_point_micros  BIGINT NOT NULL DEFAULT 0,
    version               BIGINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tenant_credit_transactions (
    id                    VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id             BIGINT NOT NULL REFERENCES tenants(id),
    account_id            VARCHAR(36) NOT NULL REFERENCES tenant_credit_accounts(id),
    type                  VARCHAR(32) NOT NULL,
    amount_point_micros   BIGINT NOT NULL,
    balance_point_micros  BIGINT NOT NULL,
    ref_no                VARCHAR(128) NOT NULL UNIQUE,
    description           TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_credit_transactions_tenant_id
    ON tenant_credit_transactions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_credit_transactions_account_id
    ON tenant_credit_transactions(account_id);

INSERT INTO billing_plans (
    id, code, name, description, edition, space_type, status, is_public,
    included_storage_bytes, included_point_micros
) VALUES
    ('plan-personal-free-v1', 'personal_free', '个人免费版', '个人工作区默认套餐', 'personal', 'personal', 'active', TRUE, 1073741824, 0),
    ('plan-personal-pro-v1', 'personal_pro', '个人专业版', '个人工作区专业套餐', 'personal', 'personal', 'active', TRUE, 21474836480, 0),
    ('plan-enterprise-team-v1', 'enterprise_team', '企业团队版', '企业工作区默认套餐', 'enterprise', 'organization', 'active', TRUE, 107374182400, 0),
    ('plan-enterprise-business-v1', 'enterprise_business', '企业商务版', '企业工作区商务套餐', 'enterprise', 'organization', 'active', TRUE, 0, 0),
    ('plan-legacy-compat-v1', 'legacy_compat', '历史兼容版', '等待后台分类的历史工作区', 'legacy', 'legacy', 'active', FALSE, 0, 0)
ON CONFLICT (code) DO NOTHING;

INSERT INTO billing_prices (
    id, plan_id, code, currency, billing_interval, amount_minor, status, is_default
) VALUES
    ('price-personal-free-cny-v1', 'plan-personal-free-v1', 'personal_free_cny_v1', 'CNY', 'none', 0, 'active', TRUE),
    ('price-personal-pro-cny-v1', 'plan-personal-pro-v1', 'personal_pro_cny_v1', 'CNY', 'month', 0, 'draft', FALSE),
    ('price-enterprise-team-cny-v1', 'plan-enterprise-team-v1', 'enterprise_team_cny_v1', 'CNY', 'month', 0, 'draft', FALSE),
    ('price-enterprise-business-cny-v1', 'plan-enterprise-business-v1', 'enterprise_business_cny_v1', 'CNY', 'month', 0, 'draft', FALSE),
    ('price-legacy-compat-cny-v1', 'plan-legacy-compat-v1', 'legacy_compat_cny_v1', 'CNY', 'none', 0, 'active', TRUE)
ON CONFLICT (code) DO NOTHING;

INSERT INTO tenant_credit_accounts (tenant_id, balance_point_micros)
SELECT t.id, GREATEST(COALESCE(t.enterprise_credits, 0), 0) * 1000000
FROM tenants t
WHERE t.deleted_at IS NULL
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO tenant_credit_transactions (
    tenant_id, account_id, type, amount_point_micros,
    balance_point_micros, ref_no, description
)
SELECT
    a.tenant_id,
    a.id,
    'legacy_import',
    a.balance_point_micros,
    a.balance_point_micros,
    'legacy-enterprise-credits:' || a.tenant_id::text,
    'Imported from tenants.enterprise_credits during billing foundation migration'
FROM tenant_credit_accounts a
WHERE a.balance_point_micros > 0
ON CONFLICT (ref_no) DO NOTHING;

INSERT INTO tenant_subscriptions (
    tenant_id, plan_id, status, billing_interval, price_snapshot, source
)
SELECT
    t.id,
    CASE
        WHEN t.space_type = 'personal' THEN 'plan-personal-free-v1'
        WHEN t.space_type = 'organization' THEN 'plan-enterprise-team-v1'
        ELSE 'plan-legacy-compat-v1'
    END,
    CASE WHEN t.space_type IN ('personal', 'organization') THEN 'active' ELSE 'legacy' END,
    'none',
    jsonb_build_object(
        'migration', '000086',
        'storage_quota_bytes', t.storage_quota,
        'legacy_enterprise_credits', COALESCE(t.enterprise_credits, 0)
    ),
    'migration'
FROM tenants t
WHERE t.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM tenant_subscriptions s
      WHERE s.tenant_id = t.id AND s.status IN ('active', 'trialing', 'legacy')
  );
