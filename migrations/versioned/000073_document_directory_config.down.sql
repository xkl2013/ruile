-- Migration: 000073_document_directory_config
-- Description: Remove manual document-directory metadata from knowledge_bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000073] Dropping directory_config from knowledge_bases'; END $$;

ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS directory_config;
