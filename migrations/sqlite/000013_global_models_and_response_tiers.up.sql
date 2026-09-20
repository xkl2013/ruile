CREATE TABLE IF NOT EXISTS system_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key VARCHAR(128) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    value_type VARCHAR(16) NOT NULL,
    category VARCHAR(32) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_secret BOOLEAN NOT NULL DEFAULT 0,
    requires_restart BOOLEAN NOT NULL DEFAULT 0,
    last_modified_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_system_settings_category
    ON system_settings(category);

UPDATE models
SET tenant_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE tenant_id <> 0;

INSERT OR IGNORE INTO system_settings (
    key,
    value,
    value_type,
    category,
    description,
    is_secret,
    requires_restart,
    last_modified_by,
    created_at,
    updated_at
)
SELECT
    'agent.response_tier_config',
    response_tier_config,
    'json',
    'agent',
    'Platform-wide fast, balanced, and ultimate model bindings.',
    0,
    0,
    '',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM tenants
WHERE response_tier_config IS NOT NULL
ORDER BY updated_at DESC, id ASC
LIMIT 1;
