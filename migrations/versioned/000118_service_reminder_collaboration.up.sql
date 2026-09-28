-- Migration 000118: service reminder collaboration and nested work items.

ALTER TABLE service_reminders
    ADD COLUMN IF NOT EXISTS parent_reminder_id VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS depth INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_service_reminders_parent
    ON service_reminders(tenant_id, service_id, parent_reminder_id, depth)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_reminder_assignees (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    reminder_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role VARCHAR(16) NOT NULL DEFAULT 'member',
    assigned_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_reminder_assignees_unique
    ON service_reminder_assignees(service_id, reminder_id, user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_reminder_assignees_scope
    ON service_reminder_assignees(tenant_id, service_id, reminder_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_reminder_comments (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    reminder_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_service_reminder_comments_scope
    ON service_reminder_comments(tenant_id, service_id, reminder_id, created_at)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_reminder_history (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    reminder_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    action VARCHAR(64) NOT NULL,
    from_status VARCHAR(32) NOT NULL DEFAULT '',
    to_status VARCHAR(32) NOT NULL DEFAULT '',
    change_detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_service_reminder_history_scope
    ON service_reminder_history(tenant_id, service_id, reminder_id, created_at DESC);
