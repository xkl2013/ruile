-- Migration 001025: immutable tenant Skill manifests with activation history.

CREATE TABLE IF NOT EXISTS tenant_skill_snapshots (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    generation INTEGER NOT NULL,
    parent_snapshot_id VARCHAR(36) NOT NULL DEFAULT '',
    state VARCHAR(24) NOT NULL,
    manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
    manifest_sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at TIMESTAMP NULL,
    superseded_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_tenant_skill_snapshots_tenant_generation
    ON tenant_skill_snapshots (tenant_id, generation DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_skill_snapshots_active
    ON tenant_skill_snapshots (tenant_id, state);
