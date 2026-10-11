-- SQLite equivalent of versioned migration 001025.

CREATE TABLE IF NOT EXISTS tenant_skill_snapshots (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    generation INTEGER NOT NULL,
    parent_snapshot_id TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    manifest TEXT NOT NULL DEFAULT '{}',
    manifest_sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at DATETIME NULL,
    superseded_at DATETIME NULL
);

CREATE INDEX IF NOT EXISTS idx_tenant_skill_snapshots_tenant_generation
    ON tenant_skill_snapshots (tenant_id, generation DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_skill_snapshots_active
    ON tenant_skill_snapshots (tenant_id, state);
