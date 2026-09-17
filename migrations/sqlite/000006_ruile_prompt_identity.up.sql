-- SQLite equivalent of versioned migration 000089.

UPDATE custom_agents
SET
    config = REPLACE(
        REPLACE(
            REPLACE(config, 'developed by Tencent', 'developed by 睿乐'),
            '由腾讯开发',
            '由睿乐开发'
        ),
        '腾讯开发的',
        '睿乐开发的'
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE deleted_at IS NULL
  AND is_builtin = 1
  AND (
      LOWER(config) LIKE '%developed by tencent%'
      OR config LIKE '%由腾讯开发%'
      OR config LIKE '%腾讯开发的%'
  );
