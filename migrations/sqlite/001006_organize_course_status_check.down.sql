PRAGMA foreign_keys = OFF;

CREATE TABLE organize_courses_old (
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

INSERT INTO organize_courses_old (
    id, tenant_id, user_id, source, title, summary, category, cover_url,
    teacher_name, teacher_title, public_status, lesson_count, learner_count,
    created_at, updated_at, deleted_at
)
SELECT
    id, tenant_id, user_id, source, title, summary, category, cover_url,
    teacher_name, teacher_title,
    CASE
        WHEN public_status IN ('pending_review', 'rejected') THEN 'draft'
        ELSE public_status
    END,
    lesson_count, learner_count, created_at, updated_at, deleted_at
FROM organize_courses;

DROP TABLE organize_courses;
ALTER TABLE organize_courses_old RENAME TO organize_courses;

CREATE INDEX IF NOT EXISTS idx_organize_courses_discover
    ON organize_courses(public_status, category, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_courses_scope_updated
    ON organize_courses(tenant_id, user_id, updated_at DESC)
    WHERE deleted_at IS NULL;

PRAGMA foreign_keys = ON;
