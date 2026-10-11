-- Migration 001020: store the admin-facing service template description.

ALTER TABLE service_space_templates
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
