-- Migration 001003: creator approval flag.
-- This was previously numbered 000027 on a parallel branch.

ALTER TABLE users ADD COLUMN is_creator BOOLEAN NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_users_is_creator ON users (is_creator);
