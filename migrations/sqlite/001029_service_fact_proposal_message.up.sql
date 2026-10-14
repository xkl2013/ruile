CREATE TABLE IF NOT EXISTS service_fact_proposals (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    source_version TEXT NOT NULL DEFAULT '',
    subject_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    needs_subject BOOLEAN NOT NULL DEFAULT 0,
    question TEXT NOT NULL DEFAULT '',
    proposal_items TEXT NOT NULL DEFAULT '[]',
    created_by TEXT NOT NULL DEFAULT '',
    resolved_by TEXT NOT NULL DEFAULT '',
    resolved_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_fact_proposals_source
    ON service_fact_proposals(tenant_id, service_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_service_fact_proposals_status
    ON service_fact_proposals(tenant_id, service_id, status, created_at DESC);

ALTER TABLE service_fact_proposals
    ADD COLUMN assistant_message_id TEXT;

CREATE INDEX IF NOT EXISTS idx_service_fact_proposals_assistant_message
    ON service_fact_proposals(tenant_id, service_id, assistant_message_id);
