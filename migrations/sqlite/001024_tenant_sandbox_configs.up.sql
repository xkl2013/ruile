-- SQLite equivalent of versioned migration 001024.

CREATE TABLE IF NOT EXISTS tenant_sandbox_configs (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    mode TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    active INTEGER NOT NULL DEFAULT 0,
    fallback_enabled INTEGER NOT NULL DEFAULT 1,
    timeout_seconds INTEGER NOT NULL DEFAULT 60,
    docker_image TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

CREATE INDEX IF NOT EXISTS idx_tenant_sandbox_configs_tenant_id
    ON tenant_sandbox_configs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_sandbox_configs_active
    ON tenant_sandbox_configs (tenant_id, active, enabled);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_sandbox_configs_name
    ON tenant_sandbox_configs (tenant_id, name);
