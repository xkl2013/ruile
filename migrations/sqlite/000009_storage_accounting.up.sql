CREATE TABLE IF NOT EXISTS tenant_storage_reservations (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    actor_user_id TEXT NOT NULL DEFAULT '',
    ref_no TEXT NOT NULL,
    operation TEXT NOT NULL,
    requested_bytes INTEGER NOT NULL DEFAULT 0 CHECK (requested_bytes >= 0),
    actual_bytes INTEGER NOT NULL DEFAULT 0 CHECK (actual_bytes >= 0),
    status TEXT NOT NULL DEFAULT 'reserved'
        CHECK (status IN ('reserved', 'committed', 'released', 'expired')),
    expires_at DATETIME NOT NULL,
    committed_at DATETIME,
    released_at DATETIME,
    failure_code TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);

CREATE INDEX IF NOT EXISTS idx_tenant_storage_reservations_active
    ON tenant_storage_reservations (tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS tenant_storage_transactions (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    actor_user_id TEXT NOT NULL DEFAULT '',
    reservation_id TEXT NOT NULL DEFAULT '',
    ref_no TEXT NOT NULL,
    operation TEXT NOT NULL,
    amount_bytes INTEGER NOT NULL,
    storage_used_after_bytes INTEGER NOT NULL CHECK (storage_used_after_bytes >= 0),
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, ref_no)
);

CREATE INDEX IF NOT EXISTS idx_tenant_storage_transactions_tenant_created
    ON tenant_storage_transactions (tenant_id, created_at DESC);
