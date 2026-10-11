-- SQLite equivalent of versioned migration 001020.

ALTER TABLE service_space_templates
    ADD COLUMN description TEXT NOT NULL DEFAULT '';
