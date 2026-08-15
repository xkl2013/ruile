-- Migration: 000074_knowledge_base_icon
-- Description: Add icon metadata to knowledge_bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000074] Adding icon to knowledge_bases'; END $$;

ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS icon TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN knowledge_bases.icon IS
    'Knowledge base display icon (TDesign icon name, emoji prefix, or image data prefix).';
