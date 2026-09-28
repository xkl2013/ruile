-- Migration 000031: service-space V2 templates, blueprints, profiles and summaries.

CREATE TABLE IF NOT EXISTS service_space_templates (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'published',
    match_rules TEXT NOT NULL DEFAULT '{}',
    blueprint TEXT NOT NULL DEFAULT '{}',
    auto_apply BOOLEAN NOT NULL DEFAULT 0,
    auto_activate BOOLEAN NOT NULL DEFAULT 0,
    risk_level TEXT NOT NULL DEFAULT 'low',
    published_by TEXT NOT NULL DEFAULT '',
    published_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_templates_version
    ON service_space_templates(tenant_id, key, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_space_templates_status
    ON service_space_templates(tenant_id, status, key);

CREATE TABLE IF NOT EXISTS service_space_blueprints (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    source_type TEXT NOT NULL,
    source_instruction TEXT NOT NULL,
    blueprint TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft',
    confirmation_mode TEXT NOT NULL DEFAULT 'pending',
    profile_version INTEGER NOT NULL DEFAULT 1,
    profile_hash TEXT NOT NULL DEFAULT '',
    confirmed_by TEXT NOT NULL DEFAULT '',
    confirmed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_blueprints_version
    ON service_space_blueprints(service_id, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_space_blueprints_current
    ON service_space_blueprints(tenant_id, service_id, status, version DESC);

CREATE TABLE IF NOT EXISTS service_space_template_applications (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    template_key TEXT NOT NULL,
    template_version INTEGER NOT NULL,
    profile_version INTEGER NOT NULL DEFAULT 1,
    profile_hash TEXT NOT NULL DEFAULT '',
    match_reason TEXT NOT NULL DEFAULT '{}',
    apply_mode TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    result TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_template_apply_key
    ON service_space_template_applications(tenant_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_service_space_template_apply_service
    ON service_space_template_applications(tenant_id, service_id, created_at DESC);

CREATE TABLE IF NOT EXISTS service_space_profiles (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    blueprint_version INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    schema_json TEXT NOT NULL DEFAULT '[]',
    profile_values TEXT NOT NULL DEFAULT '{}',
    source_watermark TEXT NOT NULL DEFAULT '',
    frozen BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_profiles_version
    ON service_space_profiles(service_id, version);

CREATE TABLE IF NOT EXISTS service_space_summaries (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    blueprint_version INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    schema_json TEXT NOT NULL DEFAULT '[]',
    sections TEXT NOT NULL DEFAULT '{}',
    source_watermark TEXT NOT NULL DEFAULT '',
    frozen BOOLEAN NOT NULL DEFAULT 0,
    refresh_status TEXT NOT NULL DEFAULT 'ready',
    error_message TEXT NOT NULL DEFAULT '',
    generated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_space_summaries_version
    ON service_space_summaries(service_id, version);
