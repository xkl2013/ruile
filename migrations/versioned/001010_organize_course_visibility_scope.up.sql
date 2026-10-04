-- Migration 001010: separate course publication from audience visibility.
--
-- Existing courses remain platform-visible so the rollout is backwards
-- compatible with the current Discover behaviour.

ALTER TABLE organize_courses
    ADD COLUMN IF NOT EXISTS visibility_scope VARCHAR(32) NOT NULL DEFAULT 'system';

UPDATE organize_courses
SET visibility_scope = 'system'
WHERE visibility_scope IS NULL OR visibility_scope = '';

ALTER TABLE organize_courses
    DROP CONSTRAINT IF EXISTS chk_organize_courses_visibility_scope;

ALTER TABLE organize_courses
    ADD CONSTRAINT chk_organize_courses_visibility_scope
    CHECK (visibility_scope IN ('system', 'shared_space', 'private'));

CREATE INDEX IF NOT EXISTS idx_organize_courses_visibility
    ON organize_courses(visibility_scope, public_status, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS organize_course_shared_spaces (
    id VARCHAR(36) PRIMARY KEY,
    course_id VARCHAR(36) NOT NULL,
    organization_id VARCHAR(36) NOT NULL,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_course_shared_spaces_unique
    ON organize_course_shared_spaces(course_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_organize_course_shared_spaces_org
    ON organize_course_shared_spaces(organization_id, course_id);
