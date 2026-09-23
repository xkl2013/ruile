-- Restore the legacy table only when explicitly rolling back this migration.
CREATE TABLE IF NOT EXISTS agent_bindings (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    profile_id TEXT NOT NULL DEFAULT '',
    agent_definition_version_id TEXT NOT NULL REFERENCES agent_definition_versions(id),
    agent_domain TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_bindings_unique
    ON agent_bindings(tenant_id, profile_id, agent_definition_version_id);
