-- Migration: 000074_knowledge_base_icon
-- Description: Remove icon metadata from knowledge_bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000074] Dropping icon from knowledge_bases'; END $$;

ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS icon;
