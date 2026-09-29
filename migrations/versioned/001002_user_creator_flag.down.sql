-- Compatibility migration moved out of the conflicting 000112 branch range.
DROP INDEX IF EXISTS idx_users_is_creator;
ALTER TABLE users DROP COLUMN IF EXISTS is_creator;
