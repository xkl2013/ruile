-- Discovery curation fields shared by public content, courses and KB
-- publications. Existing rows remain eligible for recommendation.

ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS featured BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS recommendable BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_organize_outputs_discover_curation
    ON organize_outputs(public_status, featured, recommendable, sort_order, updated_at DESC);

ALTER TABLE organize_courses ADD COLUMN IF NOT EXISTS featured BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE organize_courses ADD COLUMN IF NOT EXISTS recommendable BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE organize_courses ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_organize_courses_discover_curation
    ON organize_courses(public_status, featured, recommendable, sort_order, updated_at DESC);

ALTER TABLE knowledge_base_publications ADD COLUMN IF NOT EXISTS recommendable BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_public_kb_publications_discover_curation
    ON knowledge_base_publications(status, featured, recommendable, sort_order, updated_at DESC);
