-- Migration 001011: store source files and parser state per memory attachment.

CREATE TABLE IF NOT EXISTS organize_memory_attachments (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    memory_id VARCHAR(36) NOT NULL,
    file_name VARCHAR(512) NOT NULL,
    mime_type VARCHAR(255) NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL DEFAULT '',
    storage_url TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_stage VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    transcript TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_memory_attachments_status
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'skipped')),
    CONSTRAINT chk_organize_memory_attachments_size
        CHECK (size_bytes >= 0)
);

CREATE INDEX IF NOT EXISTS idx_organize_memory_attachments_memory
    ON organize_memory_attachments(tenant_id, user_id, memory_id, sort_order)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_memory_attachments_status
    ON organize_memory_attachments(tenant_id, user_id, status, updated_at DESC)
    WHERE deleted_at IS NULL;
