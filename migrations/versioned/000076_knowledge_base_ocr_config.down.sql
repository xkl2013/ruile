-- Migration: 000076_knowledge_base_ocr_config (rollback)
-- Description: Remove OCR fallback model configuration from knowledge bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000076] Dropping ocr_config from knowledge_bases'; END $$;

ALTER TABLE knowledge_bases
    DROP COLUMN IF EXISTS ocr_config;
