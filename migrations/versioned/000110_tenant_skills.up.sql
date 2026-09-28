CREATE TABLE IF NOT EXISTS tenant_skills (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    name VARCHAR(64) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version VARCHAR(64) NOT NULL DEFAULT '',
    bundle_path TEXT NOT NULL,
    bundle_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'ready',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_skills_name
    ON tenant_skills(tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_skills_tenant
    ON tenant_skills(tenant_id, enabled, updated_at DESC)
    WHERE deleted_at IS NULL;
