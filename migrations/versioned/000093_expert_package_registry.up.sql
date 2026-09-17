-- Migration: 000093_expert_package_registry

CREATE TABLE IF NOT EXISTS expert_packages (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id),
    package_key VARCHAR(128) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_format VARCHAR(64) NOT NULL,
    source_uri TEXT NOT NULL DEFAULT '',
    license VARCHAR(255) NOT NULL DEFAULT '',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_expert_packages_tenant_key
    ON expert_packages(tenant_id, package_key) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS expert_package_versions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    package_id VARCHAR(36) NOT NULL REFERENCES expert_packages(id),
    version VARCHAR(64) NOT NULL,
    state VARCHAR(32) NOT NULL DEFAULT 'testing',
    manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
    package_hash VARCHAR(64) NOT NULL,
    diagnostics JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    published_by VARCHAR(36) NOT NULL DEFAULT '',
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CHECK (state IN ('testing', 'published', 'deprecated', 'archived'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_expert_package_versions_immutable
    ON expert_package_versions(package_id, version) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agent_definition_versions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id),
    package_id VARCHAR(36) NOT NULL REFERENCES expert_packages(id),
    package_version_id VARCHAR(36) NOT NULL REFERENCES expert_package_versions(id),
    agent_id VARCHAR(128) NOT NULL,
    version VARCHAR(64) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    domain VARCHAR(64) NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL,
    compiled_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    skills JSONB NOT NULL DEFAULT '[]'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_contract VARCHAR(64) NOT NULL DEFAULT 'agent_result_v1',
    definition_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_definition_versions_immutable
    ON agent_definition_versions(package_version_id, agent_id) WHERE deleted_at IS NULL;

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
    ON agent_bindings(tenant_id, profile_id, agent_definition_version_id) WHERE deleted_at IS NULL;
