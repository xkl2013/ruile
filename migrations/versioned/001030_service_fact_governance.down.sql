DROP INDEX IF EXISTS idx_service_facts_active_key;

ALTER TABLE service_facts
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS disabled_at,
    DROP COLUMN IF EXISTS superseded_by,
    DROP COLUMN IF EXISTS conflict_with;
