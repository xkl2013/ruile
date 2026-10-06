-- Migration 001013: database-backed discovery category configuration.

CREATE TABLE IF NOT EXISTS organize_discover_categories (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(64) NOT NULL UNIQUE,
    label VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'enabled',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_discover_categories_status
        CHECK (status IN ('enabled', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_organize_discover_categories_listing
    ON organize_discover_categories(status, sort_order, created_at);

INSERT INTO organize_discover_categories (key, label, sort_order)
VALUES
    ('admissions_growth', '招生增长', 10),
    ('parent_service', '家长服务', 20),
    ('event_planning', '活动策划', 30),
    ('kindergarten_operations', '园所运营', 40),
    ('team_leadership', '团队运营/领导力', 50),
    ('nutrition_food_education', '儿童营养与食育', 60),
    ('space_environment', '空间设计 / 环创', 70),
    ('teacher_research', '教师成长 / 教研', 80)
ON CONFLICT (key) DO NOTHING;
