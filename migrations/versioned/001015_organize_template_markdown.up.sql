-- Migration 001015: use Markdown presets as the user-facing organize format.

ALTER TABLE organize_templates
    ADD COLUMN IF NOT EXISTS markdown_template TEXT NOT NULL DEFAULT '';
