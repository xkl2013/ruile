-- Migration 001008: materialized, versioned profiles for service subjects.

CREATE TABLE IF NOT EXISTS service_subject_profiles (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36) NOT NULL,
    blueprint_version INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    schema_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    profile_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_watermark VARCHAR(128) NOT NULL DEFAULT '',
    frozen BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_subject_profiles_version
    ON service_subject_profiles(service_id, subject_id, version);
CREATE INDEX IF NOT EXISTS idx_service_subject_profiles_current
    ON service_subject_profiles(tenant_id, service_id, subject_id, version DESC);
