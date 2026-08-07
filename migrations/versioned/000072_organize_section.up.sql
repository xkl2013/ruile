-- Migration: 000072_organize_section
-- Adds DB-backed records for the user-scoped Organize section.

CREATE TABLE IF NOT EXISTS organize_memories (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    title VARCHAR(512) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    source VARCHAR(255) NOT NULL DEFAULT '',
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_memories_kind
        CHECK (kind IN ('note', 'record', 'audio', 'audio_card')),
    CONSTRAINT chk_organize_memories_duration
        CHECK (duration_seconds >= 0)
);

CREATE INDEX IF NOT EXISTS idx_organize_memories_scope_time
    ON organize_memories(tenant_id, user_id, occurred_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_memories_scope_kind
    ON organize_memories(tenant_id, user_id, kind)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS organize_outputs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    title VARCHAR(512) NOT NULL,
    output_type VARCHAR(64) NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    source_summary VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    icon VARCHAR(64) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_outputs_status
        CHECK (status IN ('draft', 'review', 'ready', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_organize_outputs_scope_updated
    ON organize_outputs(tenant_id, user_id, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_outputs_scope_status
    ON organize_outputs(tenant_id, user_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS organize_output_memories (
    output_id VARCHAR(36) NOT NULL,
    memory_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (output_id, memory_id)
);

CREATE INDEX IF NOT EXISTS idx_organize_output_memories_scope_output
    ON organize_output_memories(tenant_id, user_id, output_id);
CREATE INDEX IF NOT EXISTS idx_organize_output_memories_scope_memory
    ON organize_output_memories(tenant_id, user_id, memory_id);

CREATE TABLE IF NOT EXISTS organize_sprout_reports (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    title VARCHAR(512) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    stage VARCHAR(64) NOT NULL DEFAULT 'organizing',
    output_hint VARCHAR(255) NOT NULL DEFAULT '',
    chips JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_sprout_reports_stage
        CHECK (stage IN ('organizing', 'expandable', 'formed'))
);

CREATE INDEX IF NOT EXISTS idx_organize_sprout_reports_scope_updated
    ON organize_sprout_reports(tenant_id, user_id, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_sprout_reports_scope_stage
    ON organize_sprout_reports(tenant_id, user_id, stage)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS organize_sprout_memories (
    report_id VARCHAR(36) NOT NULL,
    memory_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (report_id, memory_id)
);

CREATE INDEX IF NOT EXISTS idx_organize_sprout_memories_scope_report
    ON organize_sprout_memories(tenant_id, user_id, report_id);
CREATE INDEX IF NOT EXISTS idx_organize_sprout_memories_scope_memory
    ON organize_sprout_memories(tenant_id, user_id, memory_id);
