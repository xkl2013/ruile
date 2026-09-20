-- Promote the sole user-facing built-in agent override to platform scope.
-- Custom agents remain workspace-scoped. Legacy built-ins stay available from
-- the YAML registry for historical sessions but no longer keep DB overrides.

INSERT INTO custom_agents (
    id,
    name,
    description,
    avatar,
    is_builtin,
    tenant_id,
    created_by,
    runnable_by_viewer,
    config,
    created_at,
    updated_at,
    deleted_at
)
SELECT
    id,
    name,
    description,
    avatar,
    TRUE,
    0,
    created_by,
    runnable_by_viewer,
    config,
    created_at,
    CURRENT_TIMESTAMP,
    NULL
FROM custom_agents
WHERE id = 'builtin-quick-answer'
  AND is_builtin = TRUE
  AND deleted_at IS NULL
ORDER BY updated_at DESC, tenant_id ASC
LIMIT 1
ON CONFLICT (id, tenant_id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    avatar = EXCLUDED.avatar,
    is_builtin = TRUE,
    created_by = EXCLUDED.created_by,
    runnable_by_viewer = EXCLUDED.runnable_by_viewer,
    config = EXCLUDED.config,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL;

DELETE FROM custom_agents
WHERE is_builtin = TRUE
  AND tenant_id <> 0;

DELETE FROM custom_agents
WHERE is_builtin = TRUE
  AND id <> 'builtin-quick-answer';
