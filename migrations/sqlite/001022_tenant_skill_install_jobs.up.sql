-- SQLite equivalent of versioned migration 001022.

CREATE TABLE IF NOT EXISTS tenant_skill_install_jobs (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL,
    source_ref TEXT NOT NULL,
    skill_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT NOT NULL DEFAULT '',
    attempt INTEGER NOT NULL DEFAULT 1,
    cancel_requested INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_skill_install_jobs_scope
    ON tenant_skill_install_jobs(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tenant_skill_install_jobs_status
    ON tenant_skill_install_jobs(tenant_id, status, updated_at DESC);
