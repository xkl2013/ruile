-- SQLite equivalent of versioned migration 000088.

UPDATE models
SET
    is_builtin = 0,
    managed_by = '',
    updated_at = CURRENT_TIMESTAMP
WHERE is_builtin = 1 OR managed_by <> '';
