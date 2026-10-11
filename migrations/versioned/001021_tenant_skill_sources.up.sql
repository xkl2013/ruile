-- Migration 001021: persist the public source used to install a Skill.
-- This is additive so existing uploaded bundles remain runnable.

ALTER TABLE tenant_skills
    ADD COLUMN IF NOT EXISTS source_type VARCHAR(32) NOT NULL DEFAULT 'upload';

ALTER TABLE tenant_skills
    ADD COLUMN IF NOT EXISTS source_ref TEXT NOT NULL DEFAULT '';
