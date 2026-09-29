-- Migration 001004: organize course module (SQLite dialect).
--
-- Mirror of migrations/versioned/001005_organize_course_module.up.sql.
-- SQLite has no JSONB / TIMESTAMP WITH TIME ZONE, so metadata-shaped and
-- timestamp columns follow the 000000_init conventions (TEXT / DATETIME).

CREATE TABLE IF NOT EXISTS organize_courses (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
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
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (source IN ('official', 'creator')),
    CHECK (public_status IN ('draft', 'published', 'offline')),
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
    tenant_id INTEGER NOT NULL,
    output_id VARCHAR(36) NOT NULL,
    title VARCHAR(512) NOT NULL,
    lesson_type VARCHAR(16) NOT NULL DEFAULT 'article',
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (lesson_type IN ('video', 'audio', 'article')),
    CHECK (duration_seconds >= 0),
    CHECK (sort_order >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_course_lessons_order
    ON organize_course_lessons(course_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_course_lessons_output
    ON organize_course_lessons(output_id);
CREATE INDEX IF NOT EXISTS idx_organize_course_lessons_course
    ON organize_course_lessons(course_id, sort_order);
