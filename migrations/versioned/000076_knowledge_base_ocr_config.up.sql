-- Migration: 000076_knowledge_base_ocr_config
-- Description: Add OCR fallback model configuration to knowledge bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000076] Adding ocr_config to knowledge_bases'; END $$;

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS ocr_config JSONB NOT NULL DEFAULT '{}';

COMMENT ON COLUMN knowledge_bases.ocr_config IS
    'OCR fallback model configuration: {"enabled": bool, "model_id": string}';
