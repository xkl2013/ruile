DROP INDEX IF EXISTS idx_public_kb_publications_discover_curation;
ALTER TABLE knowledge_base_publications DROP COLUMN recommendable;

DROP INDEX IF EXISTS idx_organize_courses_discover_curation;
ALTER TABLE organize_courses DROP COLUMN sort_order;
ALTER TABLE organize_courses DROP COLUMN recommendable;
ALTER TABLE organize_courses DROP COLUMN featured;

DROP INDEX IF EXISTS idx_organize_outputs_discover_curation;
ALTER TABLE organize_outputs DROP COLUMN sort_order;
ALTER TABLE organize_outputs DROP COLUMN recommendable;
ALTER TABLE organize_outputs DROP COLUMN featured;
