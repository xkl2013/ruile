-- Migration 001023: store administrator-configured Skill runtime variables.

ALTER TABLE tenant_skills
    ADD COLUMN IF NOT EXISTS env_vars JSONB NOT NULL DEFAULT '{}'::jsonb;
