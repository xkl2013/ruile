DROP INDEX IF EXISTS idx_users_is_creator;
ALTER TABLE users DROP COLUMN IF EXISTS is_creator;
