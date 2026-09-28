DROP INDEX IF EXISTS idx_service_reminders_service_scope;
ALTER TABLE service_reminders DROP COLUMN IF EXISTS service_id;
