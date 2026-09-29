-- Migration 001005: organize course module.
--
-- Promotes "series" from ad-hoc columns on organize_outputs into a first-class
-- course entity, so a folder of uploaded materials can be grouped, ordered and
-- published to the discover page as a multi-lesson course.
--
-- The lesson content itself is NOT moved: organize_course_lessons.output_id
-- points at the existing organize_outputs row, which stays the single source of
-- truth for media, parsing and preview.

CREATE TABLE IF NOT EXISTS organize_courses (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    source VARCHAR(16) NOT NULL DEFAULT 'official',
    title VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    cover_url VARCHAR(512) NOT NULL DEFAULT '',
    teacher_name VARCHAR(64) NOT NULL DEFAULT '',
    teacher_title VARCHAR(128) NOT NULL DEFAULT '',
    public_status VARCHAR(32) NOT NULL DEFAULT 'published',
    lesson_count INTEGER NOT NULL DEFAULT 0,
    learner_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_courses_source
        CHECK (source IN ('official', 'creator')),
    CONSTRAINT chk_organize_courses_public_status
        CHECK (public_status IN ('draft', 'published', 'offline')),
    CONSTRAINT chk_organize_courses_counts
        CHECK (lesson_count >= 0 AND learner_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_organize_courses_discover
    ON organize_courses(public_status, category, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_courses_scope_updated
    ON organize_courses(tenant_id, user_id, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS organize_course_lessons (
    id VARCHAR(36) PRIMARY KEY,
    course_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    output_id VARCHAR(36) NOT NULL,
    title VARCHAR(512) NOT NULL,
    lesson_type VARCHAR(16) NOT NULL DEFAULT 'article',
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_organize_course_lessons_type
        CHECK (lesson_type IN ('video', 'audio', 'article')),
    CONSTRAINT chk_organize_course_lessons_duration
        CHECK (duration_seconds >= 0),
    CONSTRAINT chk_organize_course_lessons_order
        CHECK (sort_order >= 0)
);

-- One lesson per position inside a course, and one course per output: an
-- output cannot be re-grouped into a second course, which would otherwise
-- create an implicit "edit lesson 3, another course changes too" coupling.
CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_course_lessons_order
    ON organize_course_lessons(course_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_course_lessons_output
    ON organize_course_lessons(output_id);
CREATE INDEX IF NOT EXISTS idx_organize_course_lessons_course
    ON organize_course_lessons(course_id, sort_order);
