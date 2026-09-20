-- Promote the sole user-facing built-in agent override to platform scope.

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
    1,
    0,
    created_by,
    runnable_by_viewer,
    config,
    created_at,
    CURRENT_TIMESTAMP,
    NULL
FROM custom_agents
WHERE id = 'builtin-quick-answer'
  AND is_builtin = 1
  AND deleted_at IS NULL
ORDER BY updated_at DESC, tenant_id ASC
LIMIT 1
ON CONFLICT (id, tenant_id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    avatar = excluded.avatar,
    is_builtin = 1,
    created_by = excluded.created_by,
    runnable_by_viewer = excluded.runnable_by_viewer,
    config = excluded.config,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL;

DELETE FROM custom_agents
WHERE is_builtin = 1
  AND tenant_id <> 0;

DELETE FROM custom_agents
WHERE is_builtin = 1
  AND id <> 'builtin-quick-answer';
