-- SQLite equivalent of versioned migration 001018.

ALTER TABLE organize_templates
    ADD COLUMN schedule TEXT NOT NULL DEFAULT 'manual';
