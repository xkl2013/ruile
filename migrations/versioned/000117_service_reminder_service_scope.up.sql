-- Migration 000117: bind service reminders to the service space boundary.

ALTER TABLE service_reminders
    ADD COLUMN IF NOT EXISTS service_id VARCHAR(36) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_service_reminders_service_scope
    ON service_reminders(tenant_id, service_id, status, updated_at DESC)
    WHERE deleted_at IS NULL;
