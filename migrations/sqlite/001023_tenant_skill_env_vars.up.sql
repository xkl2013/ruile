-- SQLite equivalent of versioned migration 001023.

ALTER TABLE tenant_skills
    ADD COLUMN env_vars TEXT NOT NULL DEFAULT '{}';
