-- Compatibility migration moved out of the conflicting 000027 branch range.
DROP INDEX IF EXISTS idx_users_is_creator;
ALTER TABLE users DROP COLUMN is_creator;
