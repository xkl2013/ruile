-- Migration 000024: generic service-subject contract.

ALTER TABLE service_subjects ADD COLUMN service_id VARCHAR(36);
ALTER TABLE service_subjects ADD COLUMN subject_type VARCHAR(64);
ALTER TABLE service_subjects ADD COLUMN parent_subject_id VARCHAR(36);
ALTER TABLE service_subjects ADD COLUMN metadata TEXT NOT NULL DEFAULT '{}';

DROP INDEX IF EXISTS idx_service_subjects_key;

CREATE INDEX IF NOT EXISTS idx_service_subjects_service
    ON service_subjects(tenant_id, service_id, subject_type, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_service_subjects_parent
    ON service_subjects(tenant_id, service_id, parent_subject_id)
    WHERE parent_subject_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_subjects_service_key
    ON service_subjects(tenant_id, service_id, subject_key)
    WHERE service_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_subjects_legacy_key
    ON service_subjects(tenant_id, owner_user_id, subject_key)
    WHERE service_id IS NULL AND deleted_at IS NULL;

CREATE TRIGGER IF NOT EXISTS trg_service_subjects_type_insert
BEFORE INSERT ON service_subjects
WHEN NEW.service_id IS NOT NULL AND trim(COALESCE(NEW.subject_type, '')) = ''
BEGIN
    SELECT RAISE(ABORT, 'service subject type is required');
END;

CREATE TRIGGER IF NOT EXISTS trg_service_subjects_type_update
BEFORE UPDATE OF service_id, subject_type ON service_subjects
WHEN NEW.service_id IS NOT NULL AND trim(COALESCE(NEW.subject_type, '')) = ''
BEGIN
    SELECT RAISE(ABORT, 'service subject type is required');
END;
