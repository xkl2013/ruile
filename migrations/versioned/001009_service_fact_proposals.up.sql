-- Migration 001009: reviewable service-space fact proposals from chat.

CREATE TABLE IF NOT EXISTS service_fact_proposals (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    session_id VARCHAR(36) NOT NULL DEFAULT '',
    source_type VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    source_version VARCHAR(128) NOT NULL DEFAULT '',
    subject_id VARCHAR(36) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    needs_subject BOOLEAN NOT NULL DEFAULT false,
    question TEXT NOT NULL DEFAULT '',
    proposal_items JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    resolved_by VARCHAR(36) NOT NULL DEFAULT '',
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_fact_proposals_source
    ON service_fact_proposals(tenant_id, service_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_service_fact_proposals_status
    ON service_fact_proposals(tenant_id, service_id, status, created_at DESC);
