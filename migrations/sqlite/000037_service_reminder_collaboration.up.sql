-- Migration 000037: service reminder collaboration and nested work items.

ALTER TABLE service_reminders ADD COLUMN parent_reminder_id TEXT NOT NULL DEFAULT '';
ALTER TABLE service_reminders ADD COLUMN depth INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_service_reminders_parent
    ON service_reminders(tenant_id, service_id, parent_reminder_id, depth)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_reminder_assignees (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    reminder_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    assigned_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_reminder_assignees_unique
    ON service_reminder_assignees(service_id, reminder_id, user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_reminder_assignees_scope
    ON service_reminder_assignees(tenant_id, service_id, reminder_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_reminder_comments (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    reminder_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_service_reminder_comments_scope
    ON service_reminder_comments(tenant_id, service_id, reminder_id, created_at)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_reminder_history (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    reminder_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    from_status TEXT NOT NULL DEFAULT '',
    to_status TEXT NOT NULL DEFAULT '',
    change_detail TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_service_reminder_history_scope
    ON service_reminder_history(tenant_id, service_id, reminder_id, created_at DESC);
