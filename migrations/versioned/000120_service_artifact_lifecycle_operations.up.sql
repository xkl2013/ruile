-- Migration 000120: idempotent service artifact lifecycle operations.

CREATE TABLE IF NOT EXISTS service_artifact_lifecycle_operations (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    service_id VARCHAR(36) NOT NULL,
    artifact_id VARCHAR(36) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    from_lifecycle VARCHAR(32) NOT NULL,
    to_lifecycle VARCHAR(32) NOT NULL,
    result_version_id VARCHAR(36) NOT NULL DEFAULT '',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_artifact_lifecycle_ops_key
    ON service_artifact_lifecycle_operations(tenant_id, service_id, artifact_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_service_artifact_lifecycle_ops_artifact
    ON service_artifact_lifecycle_operations(tenant_id, service_id, artifact_id, created_at DESC);
