-- Migration: 000075_knowledge_base_icon_image
-- Description: Restore compact knowledge base icon storage.
DO $$ BEGIN RAISE NOTICE '[Migration 000075] Compacting knowledge_bases.icon'; END $$;

UPDATE knowledge_bases
SET icon = ''
WHERE icon LIKE 'image:%';

ALTER TABLE knowledge_bases
    ALTER COLUMN icon TYPE VARCHAR(64);

COMMENT ON COLUMN knowledge_bases.icon IS
    'Knowledge base display icon (TDesign icon name or emoji prefix).';
