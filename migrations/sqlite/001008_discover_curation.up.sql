-- Discovery curation fields shared by public content, courses and KB
-- publications. Existing rows remain eligible for recommendation.

ALTER TABLE organize_outputs ADD COLUMN featured INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_outputs ADD COLUMN recommendable INTEGER NOT NULL DEFAULT 1;
ALTER TABLE organize_outputs ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_organize_outputs_discover_curation
    ON organize_outputs(public_status, featured, recommendable, sort_order, updated_at DESC);

ALTER TABLE organize_courses ADD COLUMN featured INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_courses ADD COLUMN recommendable INTEGER NOT NULL DEFAULT 1;
ALTER TABLE organize_courses ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_organize_courses_discover_curation
    ON organize_courses(public_status, featured, recommendable, sort_order, updated_at DESC);

ALTER TABLE knowledge_base_publications ADD COLUMN recommendable INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_public_kb_publications_discover_curation
    ON knowledge_base_publications(status, featured, recommendable, sort_order, updated_at DESC);
