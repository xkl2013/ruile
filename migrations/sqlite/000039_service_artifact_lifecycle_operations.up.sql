-- Migration 000039: idempotent service artifact lifecycle operations.

CREATE TABLE IF NOT EXISTS service_artifact_lifecycle_operations (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    service_id TEXT NOT NULL,
    artifact_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    from_lifecycle TEXT NOT NULL,
    to_lifecycle TEXT NOT NULL,
    result_version_id TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_artifact_lifecycle_ops_key
    ON service_artifact_lifecycle_operations(tenant_id, service_id, artifact_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_service_artifact_lifecycle_ops_artifact
    ON service_artifact_lifecycle_operations(tenant_id, service_id, artifact_id, created_at DESC);
