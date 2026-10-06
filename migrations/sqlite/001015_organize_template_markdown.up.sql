-- SQLite equivalent of versioned migration 001015.

ALTER TABLE organize_templates
    ADD COLUMN markdown_template TEXT NOT NULL DEFAULT '';
