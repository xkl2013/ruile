-- Migration 000038: append-only structured facts for service spaces.

CREATE TABLE IF NOT EXISTS service_facts (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    subject_id TEXT NOT NULL DEFAULT '',
    fact_type TEXT NOT NULL,
    fact_key TEXT NOT NULL DEFAULT '',
    fact_value TEXT NOT NULL DEFAULT '{}',
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_version TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_service_facts_scope_created
    ON service_facts(tenant_id, service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_facts_subject_created
    ON service_facts(tenant_id, service_id, subject_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_facts_source
    ON service_facts(tenant_id, service_id, source_type, source_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_facts_source_key
    ON service_facts(tenant_id, service_id, source_type, source_id, fact_key);
