-- Migration 000109: generic service-subject contract.

ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS service_id VARCHAR(36);
ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS subject_type VARCHAR(64);
ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS parent_subject_id VARCHAR(36);
ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

DROP INDEX IF EXISTS idx_service_subjects_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_service_subjects_type_when_scoped'
    ) THEN
        ALTER TABLE service_subjects
            ADD CONSTRAINT chk_service_subjects_type_when_scoped
            CHECK (service_id IS NULL OR length(trim(COALESCE(subject_type, ''))) > 0)
            NOT VALID;
    END IF;
END $$;

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
