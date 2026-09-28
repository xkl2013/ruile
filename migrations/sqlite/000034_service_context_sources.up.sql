-- Migration 000034: persist organize outputs brought into a service space.

CREATE TABLE IF NOT EXISTS service_context_sources (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_title TEXT NOT NULL DEFAULT '',
    source_version TEXT NOT NULL DEFAULT '',
    source_summary TEXT NOT NULL DEFAULT '',
    source_content TEXT NOT NULL DEFAULT '',
    memory_ids TEXT NOT NULL DEFAULT '[]',
    metadata TEXT NOT NULL DEFAULT '{}',
    imported_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (source_type IN ('organize_output'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_context_sources_unique
    ON service_context_sources(service_id, source_type, source_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_context_sources_service
    ON service_context_sources(tenant_id, service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_context_sources_source
    ON service_context_sources(tenant_id, source_type, source_id);

