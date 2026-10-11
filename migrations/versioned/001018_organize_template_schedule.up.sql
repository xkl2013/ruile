-- Migration 001018: allow platform organize templates to define a default execution cycle.

ALTER TABLE organize_templates
    ADD COLUMN IF NOT EXISTS schedule VARCHAR(32) NOT NULL DEFAULT 'manual';
