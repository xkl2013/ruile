-- Migration 000105: service spaces, service-scoped sessions, and artifact index.

CREATE TABLE IF NOT EXISTS services (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    owner_user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    instruction TEXT NOT NULL DEFAULT '',
    knowledge_base_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    selected_skills JSONB NOT NULL DEFAULT '[]'::jsonb,
    campus_scope JSONB NOT NULL DEFAULT '[]'::jsonb,
    course_scope JSONB NOT NULL DEFAULT '[]'::jsonb,
    template_key VARCHAR(64) NOT NULL DEFAULT '',
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    is_default BOOLEAN NOT NULL DEFAULT false,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private',
    member_limit INTEGER NOT NULL DEFAULT 20,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    migrated_from_profile_id VARCHAR(36),
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    updated_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_services_state
        CHECK (state IN ('draft', 'active', 'paused', 'archived')),
    CONSTRAINT chk_services_visibility
        CHECK (visibility IN ('private', 'tenant'))
);

CREATE INDEX IF NOT EXISTS idx_services_owner
    ON services(tenant_id, owner_user_id, state)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_services_tenant_state
    ON services(tenant_id, state, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_default
    ON services(tenant_id, owner_user_id)
    WHERE is_default = true AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_members (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'viewer',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    invited_by VARCHAR(36) NOT NULL DEFAULT '',
    joined_at TIMESTAMP WITH TIME ZONE,
    left_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_service_members_role
        CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    CONSTRAINT chk_service_members_status
        CHECK (status IN ('active', 'left'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_members_unique
    ON service_members(service_id, user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_members_user
    ON service_members(tenant_id, user_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_expert_bindings (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    expert_ref VARCHAR(128) NOT NULL,
    expert_name VARCHAR(255) NOT NULL DEFAULT '',
    expert_domain VARCHAR(64) NOT NULL DEFAULT '',
    source VARCHAR(32) NOT NULL DEFAULT 'builtin',
    enabled BOOLEAN NOT NULL DEFAULT true,
    display_order INTEGER NOT NULL DEFAULT 0,
    instruction_override TEXT NOT NULL DEFAULT '',
    work_doc_directory VARCHAR(255) NOT NULL DEFAULT '',
    output_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    memory_filter JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    updated_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_expert_bindings_unique
    ON service_expert_bindings(service_id, expert_ref)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_expert_bindings_service
    ON service_expert_bindings(tenant_id, service_id, display_order)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_artifacts (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36) NOT NULL DEFAULT '',
    run_id VARCHAR(36) NOT NULL DEFAULT '',
    work_doc_id VARCHAR(36) NOT NULL DEFAULT '',
    artifact_id VARCHAR(36) NOT NULL,
    version_id VARCHAR(36) NOT NULL,
    kind VARCHAR(64) NOT NULL DEFAULT 'document',
    format VARCHAR(64) NOT NULL DEFAULT '',
    title VARCHAR(512) NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    resource_id VARCHAR(36) NOT NULL DEFAULT '',
    resource_ref TEXT NOT NULL DEFAULT '',
    mime_type VARCHAR(255) NOT NULL DEFAULT '',
    original_name VARCHAR(1024) NOT NULL DEFAULT '',
    lifecycle VARCHAR(32) NOT NULL DEFAULT 'saved',
    version INTEGER NOT NULL DEFAULT 1,
    is_current BOOLEAN NOT NULL DEFAULT true,
    previewable BOOLEAN NOT NULL DEFAULT false,
    downloadable BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_service_artifacts_lifecycle
        CHECK (lifecycle IN ('temporary', 'saved', 'shared', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_artifacts_version
    ON service_artifacts(service_id, version_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_artifacts_current
    ON service_artifacts(tenant_id, service_id, is_current, updated_at DESC)
    WHERE deleted_at IS NULL;

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS service_id VARCHAR(36);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS expert_ref VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS expert_name VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_service
    ON sessions(tenant_id, service_id, is_pinned DESC, pinned_at DESC, updated_at DESC)
    WHERE service_id IS NOT NULL AND deleted_at IS NULL;

ALTER TABLE agent_runs ADD COLUMN IF NOT EXISTS service_id VARCHAR(36) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_agent_runs_service
    ON agent_runs(tenant_id, service_id, created_at DESC)
    WHERE service_id <> '' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_runs_service_thread
    ON agent_runs(tenant_id, service_id, thread_id, created_at)
    WHERE service_id <> '' AND deleted_at IS NULL;
