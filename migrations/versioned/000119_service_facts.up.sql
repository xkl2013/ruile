-- Migration 000119: append-only structured facts for service spaces.

CREATE TABLE IF NOT EXISTS service_facts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36) NOT NULL DEFAULT '',
    fact_type VARCHAR(64) NOT NULL,
    fact_key VARCHAR(128) NOT NULL DEFAULT '',
    fact_value JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_type VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    source_version VARCHAR(128) NOT NULL DEFAULT '',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_service_facts_scope_created
    ON service_facts(tenant_id, service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_facts_subject_created
    ON service_facts(tenant_id, service_id, subject_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_service_facts_source
    ON service_facts(tenant_id, service_id, source_type, source_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_facts_source_key
    ON service_facts(tenant_id, service_id, source_type, source_id, fact_key);
