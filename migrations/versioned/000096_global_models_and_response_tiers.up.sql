-- Promote model configuration and answer tiers from workspace scope to
-- platform scope. Model IDs are already globally unique, so references stored
-- by knowledge bases, agents, and sessions remain valid.

UPDATE models
SET tenant_id = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE tenant_id <> 0;

INSERT INTO system_settings (
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
    FALSE,
    FALSE,
    '',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM tenants
WHERE response_tier_config IS NOT NULL
ORDER BY updated_at DESC, id ASC
LIMIT 1
ON CONFLICT (key) DO NOTHING;
