DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS task_dead_letters;
DROP TABLE IF EXISTS task_pending_ops;

ALTER TABLE messages DROP COLUMN attachments;
