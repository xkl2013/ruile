CREATE TABLE IF NOT EXISTS tenant_skills (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    bundle_path TEXT NOT NULL,
    bundle_sha256 TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ready',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_skills_name
    ON tenant_skills(tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_skills_tenant
    ON tenant_skills(tenant_id, enabled, updated_at DESC);
