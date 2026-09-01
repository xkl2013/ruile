-- Migration: 000078_service_module
-- Adds the service-reminder module: work profiles, agent settings, service subjects,
-- markdown work docs, evidence links, reminders and action drafts.

CREATE TABLE IF NOT EXISTS user_work_profiles (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role_type VARCHAR(64) NOT NULL DEFAULT '',
    campus_scope JSONB NOT NULL DEFAULT '[]'::jsonb,
    course_scope JSONB NOT NULL DEFAULT '[]'::jsonb,
    memory_scope TEXT NOT NULL DEFAULT '',
    tone_preference VARCHAR(255) NOT NULL DEFAULT '',
    default_profile BOOLEAN NOT NULL DEFAULT false,
    enabled BOOLEAN NOT NULL DEFAULT false,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    updated_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_user_work_profiles_state
        CHECK (state IN ('draft', 'testing', 'enabled', 'disabled', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_user_work_profiles_scope
    ON user_work_profiles(tenant_id, user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_work_profiles_state
    ON user_work_profiles(tenant_id, state, enabled)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_work_profiles_default
    ON user_work_profiles(tenant_id, user_id)
    WHERE default_profile = true AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS work_profile_agent_settings (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    profile_id VARCHAR(36) NOT NULL,
    agent_id VARCHAR(64) NOT NULL DEFAULT '',
    agent_domain VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0,
    memory_filter JSONB NOT NULL DEFAULT '{}'::jsonb,
    knowledge_base_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    work_doc_directory VARCHAR(255) NOT NULL DEFAULT '',
    selected_skills JSONB NOT NULL DEFAULT '[]'::jsonb,
    output_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    updated_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_work_profile_agent_settings_domain
        CHECK (agent_domain IN ('memory_router', 'lead_intake', 'sales_consulting', 'customer_service', 'schedule_coordination', 'after_sale_risk', 'daily_review'))
);

CREATE INDEX IF NOT EXISTS idx_work_profile_agent_settings_profile
    ON work_profile_agent_settings(tenant_id, profile_id, display_order)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_work_profile_agent_settings_enabled
    ON work_profile_agent_settings(tenant_id, profile_id, enabled)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_work_profile_agent_settings_domain
    ON work_profile_agent_settings(tenant_id, profile_id, agent_domain)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS service_subjects (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    owner_user_id VARCHAR(36) NOT NULL,
    subject_key VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    student_name VARCHAR(255) NOT NULL DEFAULT '',
    relation VARCHAR(64) NOT NULL DEFAULT '',
    aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    external_refs JSONB NOT NULL DEFAULT '{}'::jsonb,
    visibility_scope VARCHAR(64) NOT NULL DEFAULT 'private',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_service_subjects_owner
    ON service_subjects(tenant_id, owner_user_id)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_subjects_key
    ON service_subjects(tenant_id, owner_user_id, subject_key)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_work_docs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    profile_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36) NOT NULL,
    owner_user_id VARCHAR(36) NOT NULL,
    agent_domain VARCHAR(64) NOT NULL,
    doc_type VARCHAR(64) NOT NULL DEFAULT 'customer_workspace',
    doc_path VARCHAR(512) NOT NULL,
    title VARCHAR(512) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'current',
    source_memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_agent_work_docs_status
        CHECK (status IN ('current', 'stale', 'recompute_required', 'archived', 'hidden_due_to_source_permission'))
);

CREATE INDEX IF NOT EXISTS idx_agent_work_docs_subject
    ON agent_work_docs(tenant_id, profile_id, subject_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_work_docs_path
    ON agent_work_docs(tenant_id, doc_path)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_work_docs_unique_path
    ON agent_work_docs(tenant_id, profile_id, subject_id, agent_domain, doc_path)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_work_doc_memory_links (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    doc_id VARCHAR(36) NOT NULL,
    doc_path VARCHAR(512) NOT NULL,
    memory_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36) NOT NULL,
    agent_domain VARCHAR(64) NOT NULL,
    link_type VARCHAR(32) NOT NULL DEFAULT 'evidence',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    evidence_excerpt TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_agent_work_doc_memory_links_type
        CHECK (link_type IN ('trigger', 'evidence', 'follow_up', 'risk', 'resolved'))
);

CREATE INDEX IF NOT EXISTS idx_agent_work_doc_memory_links_doc
    ON agent_work_doc_memory_links(tenant_id, doc_id);
CREATE INDEX IF NOT EXISTS idx_agent_work_doc_memory_links_memory
    ON agent_work_doc_memory_links(tenant_id, memory_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_work_doc_memory_links_unique
    ON agent_work_doc_memory_links(tenant_id, doc_id, memory_id, link_type);

CREATE TABLE IF NOT EXISTS service_reminders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    profile_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36) NOT NULL,
    agent_domain VARCHAR(64) NOT NULL,
    title VARCHAR(512) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    priority VARCHAR(16) NOT NULL DEFAULT 'low',
    due_at TIMESTAMP WITH TIME ZONE,
    due_text VARCHAR(64) NOT NULL DEFAULT '',
    stage VARCHAR(128) NOT NULL DEFAULT '',
    channel VARCHAR(128) NOT NULL DEFAULT '',
    decision_role VARCHAR(128) NOT NULL DEFAULT '',
    risk_label VARCHAR(128) NOT NULL DEFAULT '',
    assist_reason TEXT NOT NULL DEFAULT '',
    primary_action TEXT NOT NULL DEFAULT '',
    next_action TEXT NOT NULL DEFAULT '',
    avoid_action TEXT NOT NULL DEFAULT '',
    context_items JSONB NOT NULL DEFAULT '[]'::jsonb,
    memory_signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_memory_count INTEGER NOT NULL DEFAULT 0,
    last_memory_at TIMESTAMP WITH TIME ZONE,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    sales_highlights JSONB NOT NULL DEFAULT '[]'::jsonb,
    write_back_status VARCHAR(64) NOT NULL DEFAULT '',
    write_back_draft TEXT NOT NULL DEFAULT '',
    reply_draft TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_service_reminders_status
        CHECK (status IN ('candidate', 'pending', 'generated', 'confirmed', 'completed', 'ignored', 'snoozed', 'stale', 'recompute_required')),
    CONSTRAINT chk_service_reminders_priority
        CHECK (priority IN ('high', 'medium', 'low'))
);

CREATE INDEX IF NOT EXISTS idx_service_reminders_scope
    ON service_reminders(tenant_id, user_id, profile_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_reminders_status
    ON service_reminders(tenant_id, user_id, status)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_reminders_due
    ON service_reminders(tenant_id, user_id, priority, due_at)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_action_drafts (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    reminder_id VARCHAR(36) NOT NULL,
    agent_id VARCHAR(64) NOT NULL DEFAULT '',
    agent_domain VARCHAR(64) NOT NULL,
    action_type VARCHAR(64) NOT NULL DEFAULT 'follow_up',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    title VARCHAR(512) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    external_system VARCHAR(128) NOT NULL DEFAULT '',
    external_object_id VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_agent_action_drafts_status
        CHECK (status IN ('draft', 'confirmed', 'executing', 'succeeded', 'ignored', 'snoozed', 'failed', 'retryable'))
);

CREATE INDEX IF NOT EXISTS idx_agent_action_drafts_reminder
    ON agent_action_drafts(tenant_id, user_id, reminder_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_action_drafts_status
    ON agent_action_drafts(tenant_id, user_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_action_logs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL,
    action_draft_id VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_action_logs_draft
    ON agent_action_logs(tenant_id, action_draft_id, created_at DESC);
