-- SQLite equivalent of versioned migration 001021.

ALTER TABLE tenant_skills
    ADD COLUMN source_type TEXT NOT NULL DEFAULT 'upload';

ALTER TABLE tenant_skills
    ADD COLUMN source_ref TEXT NOT NULL DEFAULT '';
