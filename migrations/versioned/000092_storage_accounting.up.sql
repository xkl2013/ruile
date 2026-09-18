CREATE TABLE IF NOT EXISTS tenant_storage_reservations (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
    ref_no VARCHAR(160) NOT NULL,
    operation VARCHAR(32) NOT NULL,
    requested_bytes BIGINT NOT NULL DEFAULT 0 CHECK (requested_bytes >= 0),
    actual_bytes BIGINT NOT NULL DEFAULT 0 CHECK (actual_bytes >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'reserved'
        CHECK (status IN ('reserved', 'committed', 'released', 'expired')),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    committed_at TIMESTAMP WITH TIME ZONE,
    released_at TIMESTAMP WITH TIME ZONE,
    failure_code VARCHAR(64) NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);

CREATE INDEX IF NOT EXISTS idx_tenant_storage_reservations_active
    ON tenant_storage_reservations (tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS tenant_storage_transactions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
    reservation_id VARCHAR(36) NOT NULL DEFAULT '',
    ref_no VARCHAR(160) NOT NULL,
    operation VARCHAR(32) NOT NULL,
    amount_bytes BIGINT NOT NULL,
    storage_used_after_bytes BIGINT NOT NULL CHECK (storage_used_after_bytes >= 0),
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);

CREATE INDEX IF NOT EXISTS idx_tenant_storage_transactions_tenant_created
    ON tenant_storage_transactions (tenant_id, created_at DESC);
