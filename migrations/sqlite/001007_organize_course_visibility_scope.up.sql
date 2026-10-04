-- Migration 001007: separate course publication from audience visibility.
-- Existing courses remain visible to every authenticated Discover viewer.

ALTER TABLE organize_courses
    ADD COLUMN visibility_scope VARCHAR(32) NOT NULL DEFAULT 'system';

UPDATE organize_courses
SET visibility_scope = 'system'
WHERE visibility_scope IS NULL OR visibility_scope = '';

CREATE INDEX IF NOT EXISTS idx_organize_courses_visibility
    ON organize_courses(visibility_scope, public_status, updated_at DESC);

CREATE TABLE IF NOT EXISTS organize_course_shared_spaces (
    id VARCHAR(36) PRIMARY KEY,
    course_id VARCHAR(36) NOT NULL,
    organization_id VARCHAR(36) NOT NULL,
    created_by VARCHAR(36) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_course_shared_spaces_unique
    ON organize_course_shared_spaces(course_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_organize_course_shared_spaces_org
    ON organize_course_shared_spaces(organization_id, course_id);
