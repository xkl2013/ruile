-- Model ownership cannot be reconstructed after promotion because one model
-- may already be referenced by multiple workspaces. Keep models system-scoped
-- and only remove the platform answer-tier override.
DELETE FROM system_settings
WHERE key = 'agent.response_tier_config';
