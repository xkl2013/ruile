-- Migration 000112: service-space V2 templates, blueprints, profiles and summaries.

CREATE TABLE IF NOT EXISTS service_space_templates (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    key VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'published',
    match_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    blueprint JSONB NOT NULL DEFAULT '{}'::jsonb,
    auto_apply BOOLEAN NOT NULL DEFAULT false,
    auto_activate BOOLEAN NOT NULL DEFAULT false,
    risk_level VARCHAR(16) NOT NULL DEFAULT 'low',
    published_by VARCHAR(36) NOT NULL DEFAULT '',
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_templates_version
    ON service_space_templates(tenant_id, key, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_space_templates_status
    ON service_space_templates(tenant_id, status, key) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_space_blueprints (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    source_type VARCHAR(32) NOT NULL,
    source_instruction TEXT NOT NULL,
    blueprint JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    confirmation_mode VARCHAR(32) NOT NULL DEFAULT 'pending',
    profile_version INTEGER NOT NULL DEFAULT 1,
    profile_hash VARCHAR(64) NOT NULL DEFAULT '',
    confirmed_by VARCHAR(36) NOT NULL DEFAULT '',
    confirmed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_blueprints_version
    ON service_space_blueprints(service_id, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_space_blueprints_current
    ON service_space_blueprints(tenant_id, service_id, status, version DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_space_template_applications (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    template_key VARCHAR(128) NOT NULL,
    template_version INTEGER NOT NULL,
    profile_version INTEGER NOT NULL DEFAULT 1,
    profile_hash VARCHAR(64) NOT NULL DEFAULT '',
    match_reason JSONB NOT NULL DEFAULT '{}'::jsonb,
    apply_mode VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    result VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_template_apply_key
    ON service_space_template_applications(tenant_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_service_space_template_apply_service
    ON service_space_template_applications(tenant_id, service_id, created_at DESC);

CREATE TABLE IF NOT EXISTS service_space_profiles (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    blueprint_version INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    schema_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    profile_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_watermark VARCHAR(128) NOT NULL DEFAULT '',
    frozen BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_profiles_version
    ON service_space_profiles(service_id, version);

CREATE TABLE IF NOT EXISTS service_space_summaries (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    blueprint_version INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    schema_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    sections JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_watermark VARCHAR(128) NOT NULL DEFAULT '',
    frozen BOOLEAN NOT NULL DEFAULT false,
    refresh_status VARCHAR(32) NOT NULL DEFAULT 'ready',
    error_message TEXT NOT NULL DEFAULT '',
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_summaries_version
    ON service_space_summaries(service_id, version);
