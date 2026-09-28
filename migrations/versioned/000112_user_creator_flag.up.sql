-- Mark platform-approved content creators explicitly.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_creator BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_is_creator ON users (is_creator);

COMMENT ON COLUMN users.is_creator IS 'Whether the user is approved by a SystemAdmin to publish platform content';
