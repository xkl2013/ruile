-- Remove upstream third-party developer attribution from persisted built-in
-- agent prompts. Built-in agents stored in custom_agents take precedence over
-- YAML defaults, so existing installations need this data migration.

UPDATE custom_agents
SET
    config = REPLACE(
        REPLACE(
            REPLACE(config::TEXT, 'developed by Tencent', 'developed by 睿乐'),
            '由腾讯开发',
            '由睿乐开发'
        ),
        '腾讯开发的',
        '睿乐开发的'
    )::JSONB,
    updated_at = CURRENT_TIMESTAMP
WHERE deleted_at IS NULL
  AND is_builtin = TRUE
  AND (
      config::TEXT ILIKE '%developed by Tencent%'
      OR config::TEXT LIKE '%由腾讯开发%'
      OR config::TEXT LIKE '%腾讯开发的%'
  );
