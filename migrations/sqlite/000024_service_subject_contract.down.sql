BEGIN;

DROP TRIGGER IF EXISTS trg_service_subjects_type_update;
DROP TRIGGER IF EXISTS trg_service_subjects_type_insert;
DROP INDEX IF EXISTS idx_service_subjects_service_key;
DROP INDEX IF EXISTS idx_service_subjects_parent;
DROP INDEX IF EXISTS idx_service_subjects_service;
DROP INDEX IF EXISTS idx_service_subjects_legacy_key;

-- Preflight: fail before dropping columns if new service-scoped data cannot
-- satisfy the legacy owner-scoped uniqueness contract on rollback.
CREATE UNIQUE INDEX idx_service_subjects_key
    ON service_subjects(tenant_id, owner_user_id, subject_key)
    WHERE deleted_at IS NULL;

ALTER TABLE service_subjects DROP COLUMN metadata;
ALTER TABLE service_subjects DROP COLUMN parent_subject_id;
ALTER TABLE service_subjects DROP COLUMN subject_type;
ALTER TABLE service_subjects DROP COLUMN service_id;

COMMIT;
