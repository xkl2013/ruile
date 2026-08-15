-- Migration: 000075_knowledge_base_icon_image
-- Description: Allow knowledge base icons to store uploaded image data.
DO $$ BEGIN RAISE NOTICE '[Migration 000075] Expanding knowledge_bases.icon for image icons'; END $$;

ALTER TABLE knowledge_bases
    ALTER COLUMN icon TYPE TEXT;

COMMENT ON COLUMN knowledge_bases.icon IS
    'Knowledge base display icon (TDesign icon name, emoji prefix, or image data prefix).';
