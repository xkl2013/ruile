-- Migration 000020: service spaces, service-scoped sessions, and artifact index.

CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    owner_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    instruction TEXT NOT NULL DEFAULT '',
    knowledge_base_ids TEXT NOT NULL DEFAULT '[]',
    selected_skills TEXT NOT NULL DEFAULT '[]',
    campus_scope TEXT NOT NULL DEFAULT '[]',
    course_scope TEXT NOT NULL DEFAULT '[]',
    template_key TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT 'draft',
    is_default BOOLEAN NOT NULL DEFAULT 0,
    visibility TEXT NOT NULL DEFAULT 'private',
    member_limit INTEGER NOT NULL DEFAULT 20,
    settings TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    migrated_from_profile_id TEXT,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (state IN ('draft', 'active', 'paused', 'archived')),
    CHECK (visibility IN ('private', 'tenant'))
);
CREATE INDEX IF NOT EXISTS idx_services_owner ON services(tenant_id, owner_user_id, state);
CREATE INDEX IF NOT EXISTS idx_services_tenant_state ON services(tenant_id, state, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_default
    ON services(tenant_id, owner_user_id) WHERE is_default = 1 AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_members (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer',
    status TEXT NOT NULL DEFAULT 'active',
    invited_by TEXT NOT NULL DEFAULT '',
    joined_at DATETIME,
    left_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    CHECK (status IN ('active', 'left'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_members_unique
    ON service_members(service_id, user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_members_user ON service_members(tenant_id, user_id);

CREATE TABLE IF NOT EXISTS service_expert_bindings (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    expert_ref TEXT NOT NULL,
    expert_name TEXT NOT NULL DEFAULT '',
    expert_domain TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'builtin',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    display_order INTEGER NOT NULL DEFAULT 0,
    instruction_override TEXT NOT NULL DEFAULT '',
    work_doc_directory TEXT NOT NULL DEFAULT '',
    output_policy TEXT NOT NULL DEFAULT '{}',
    memory_filter TEXT NOT NULL DEFAULT '{}',
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_expert_bindings_unique
    ON service_expert_bindings(service_id, expert_ref) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_expert_bindings_service
    ON service_expert_bindings(tenant_id, service_id, display_order);

CREATE TABLE IF NOT EXISTS service_artifacts (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    subject_id TEXT NOT NULL DEFAULT '',
    run_id TEXT NOT NULL DEFAULT '',
    work_doc_id TEXT NOT NULL DEFAULT '',
    artifact_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'document',
    format TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    resource_ref TEXT NOT NULL DEFAULT '',
    mime_type TEXT NOT NULL DEFAULT '',
    original_name TEXT NOT NULL DEFAULT '',
    lifecycle TEXT NOT NULL DEFAULT 'saved',
    version INTEGER NOT NULL DEFAULT 1,
    is_current BOOLEAN NOT NULL DEFAULT 1,
    previewable BOOLEAN NOT NULL DEFAULT 0,
    downloadable BOOLEAN NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (lifecycle IN ('temporary', 'saved', 'shared', 'archived'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_artifacts_version
    ON service_artifacts(service_id, version_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_artifacts_current
    ON service_artifacts(tenant_id, service_id, is_current, updated_at DESC);

ALTER TABLE sessions ADD COLUMN service_id TEXT;
ALTER TABLE sessions ADD COLUMN expert_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN expert_name TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_service
    ON sessions(tenant_id, service_id, is_pinned DESC, pinned_at DESC, updated_at DESC);

ALTER TABLE agent_runs ADD COLUMN service_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_agent_runs_service
    ON agent_runs(tenant_id, service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_service_thread
    ON agent_runs(tenant_id, service_id, thread_id, created_at);
