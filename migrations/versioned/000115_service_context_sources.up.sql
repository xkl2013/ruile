-- Migration 000115: persist organize outputs brought into a service space.

CREATE TABLE IF NOT EXISTS service_context_sources (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    source_type VARCHAR(32) NOT NULL,
    source_id VARCHAR(36) NOT NULL,
    source_title VARCHAR(512) NOT NULL DEFAULT '',
    source_version VARCHAR(64) NOT NULL DEFAULT '',
    source_summary TEXT NOT NULL DEFAULT '',
    source_content TEXT NOT NULL DEFAULT '',
    memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    imported_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CHECK (source_type IN ('organize_output'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_context_sources_unique
    ON service_context_sources(service_id, source_type, source_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_context_sources_service
    ON service_context_sources(tenant_id, service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_context_sources_source
    ON service_context_sources(tenant_id, source_type, source_id);

