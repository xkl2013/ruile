-- Migration 001030: add conflict, supersession, disable, and soft-delete governance.

ALTER TABLE service_facts
    ADD COLUMN IF NOT EXISTS conflict_with VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS superseded_by VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_service_facts_active_key
    ON service_facts(tenant_id, service_id, subject_id, fact_key, created_at DESC)
    WHERE disabled_at IS NULL AND deleted_at IS NULL AND superseded_by = '';
