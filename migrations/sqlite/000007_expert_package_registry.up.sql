CREATE TABLE IF NOT EXISTS expert_packages (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    package_key TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_format TEXT NOT NULL,
    source_uri TEXT NOT NULL DEFAULT '',
    license TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_expert_packages_tenant_key
    ON expert_packages(tenant_id, package_key);

CREATE TABLE IF NOT EXISTS expert_package_versions (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES expert_packages(id),
    version TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'testing',
    manifest TEXT NOT NULL DEFAULT '{}',
    package_hash TEXT NOT NULL,
    diagnostics TEXT NOT NULL DEFAULT '{}',
    created_by TEXT NOT NULL DEFAULT '',
    published_by TEXT NOT NULL DEFAULT '',
    published_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (state IN ('testing', 'published', 'deprecated', 'archived'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_expert_package_versions_immutable
    ON expert_package_versions(package_id, version);

CREATE TABLE IF NOT EXISTS agent_definition_versions (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    package_id TEXT NOT NULL REFERENCES expert_packages(id),
    package_version_id TEXT NOT NULL REFERENCES expert_package_versions(id),
    agent_id TEXT NOT NULL,
    version TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    domain TEXT NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL,
    compiled_config TEXT NOT NULL DEFAULT '{}',
    skills TEXT NOT NULL DEFAULT '[]',
    capabilities TEXT NOT NULL DEFAULT '{}',
    output_contract TEXT NOT NULL DEFAULT 'agent_result_v1',
    definition_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_definition_versions_immutable
    ON agent_definition_versions(package_version_id, agent_id);

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
