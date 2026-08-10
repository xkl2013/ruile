-- Migration: 000073_document_directory_config
-- Description: Persist manual document-directory metadata on knowledge_bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000073] Adding directory_config to knowledge_bases'; END $$;

ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS directory_config JSONB DEFAULT NULL;

COMMENT ON COLUMN knowledge_bases.directory_config IS
    'Manual document-directory state for the KB UI: {"root_description": string, "directories": [{"path": string, "name": string, "description": string, "parent_path": string, "created_at": string, "updated_at": string}]}';
