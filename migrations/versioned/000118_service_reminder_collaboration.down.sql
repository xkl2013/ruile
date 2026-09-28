DROP INDEX IF EXISTS idx_service_reminder_history_scope;
DROP TABLE IF EXISTS service_reminder_history;
DROP INDEX IF EXISTS idx_service_reminder_comments_scope;
DROP TABLE IF EXISTS service_reminder_comments;
DROP INDEX IF EXISTS idx_service_reminder_assignees_scope;
DROP INDEX IF EXISTS idx_service_reminder_assignees_unique;
DROP TABLE IF EXISTS service_reminder_assignees;
DROP INDEX IF EXISTS idx_service_reminders_parent;
ALTER TABLE service_reminders
    DROP COLUMN IF EXISTS parent_reminder_id,
    DROP COLUMN IF EXISTS depth;
