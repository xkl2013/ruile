-- Migration 000036: bind service reminders to the service space boundary.

ALTER TABLE service_reminders
    ADD COLUMN service_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_service_reminders_service_scope
    ON service_reminders(tenant_id, service_id, status, updated_at);
