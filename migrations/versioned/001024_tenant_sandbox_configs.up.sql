-- Migration 001024: persist tenant-scoped sandbox policies for Agent skill execution.

CREATE TABLE IF NOT EXISTS tenant_sandbox_configs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    mode VARCHAR(16) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    fallback_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    timeout_seconds INTEGER NOT NULL DEFAULT 60,
    docker_image VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_tenant_sandbox_configs_tenant_id
    ON tenant_sandbox_configs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_sandbox_configs_active
    ON tenant_sandbox_configs (tenant_id, active, enabled);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_sandbox_configs_name
    ON tenant_sandbox_configs (tenant_id, name);
