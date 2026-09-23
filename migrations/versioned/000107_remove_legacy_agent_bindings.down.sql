-- Restore the legacy table only when explicitly rolling back this migration.
CREATE TABLE IF NOT EXISTS agent_bindings (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id),
    profile_id VARCHAR(36) NOT NULL DEFAULT '',
    agent_definition_version_id VARCHAR(36) NOT NULL REFERENCES agent_definition_versions(id),
    agent_domain VARCHAR(64) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_bindings_unique
    ON agent_bindings(tenant_id, profile_id, agent_definition_version_id)
    WHERE deleted_at IS NULL;
