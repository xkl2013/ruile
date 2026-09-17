-- Retire platform-wide built-in models. Existing rows remain in their
-- original workspace so administrators can manage or remove them normally.

UPDATE models
SET
    is_builtin = FALSE,
    managed_by = '',
    updated_at = CURRENT_TIMESTAMP
WHERE is_builtin = TRUE OR managed_by <> '';
