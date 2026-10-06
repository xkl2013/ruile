DROP INDEX IF EXISTS idx_public_kb_publications_discover_curation;
ALTER TABLE knowledge_base_publications DROP COLUMN IF EXISTS recommendable;

DROP INDEX IF EXISTS idx_organize_courses_discover_curation;
ALTER TABLE organize_courses DROP COLUMN IF EXISTS sort_order;
ALTER TABLE organize_courses DROP COLUMN IF EXISTS recommendable;
ALTER TABLE organize_courses DROP COLUMN IF EXISTS featured;

DROP INDEX IF EXISTS idx_organize_outputs_discover_curation;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS sort_order;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS recommendable;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS featured;
