-- Migration 001022: persist asynchronous Skill source installation attempts.

CREATE TABLE IF NOT EXISTS tenant_skill_install_jobs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    source_type VARCHAR(32) NOT NULL,
    source_ref TEXT NOT NULL,
    skill_id VARCHAR(36) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    error TEXT NOT NULL DEFAULT '',
    attempt INTEGER NOT NULL DEFAULT 1,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_skill_install_jobs_scope
    ON tenant_skill_install_jobs(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tenant_skill_install_jobs_status
    ON tenant_skill_install_jobs(tenant_id, status, updated_at DESC);
