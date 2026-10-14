DROP INDEX IF EXISTS idx_service_facts_active_key;

ALTER TABLE service_facts DROP COLUMN deleted_at;
ALTER TABLE service_facts DROP COLUMN disabled_at;
ALTER TABLE service_facts DROP COLUMN superseded_by;
ALTER TABLE service_facts DROP COLUMN conflict_with;
