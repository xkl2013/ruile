ALTER TABLE service_facts ADD COLUMN conflict_with TEXT NOT NULL DEFAULT '';
ALTER TABLE service_facts ADD COLUMN superseded_by TEXT NOT NULL DEFAULT '';
ALTER TABLE service_facts ADD COLUMN disabled_at DATETIME;
ALTER TABLE service_facts ADD COLUMN deleted_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_service_facts_active_key
    ON service_facts(tenant_id, service_id, subject_id, fact_key, created_at DESC);
