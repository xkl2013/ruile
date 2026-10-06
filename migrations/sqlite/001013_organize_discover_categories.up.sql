-- SQLite equivalent of versioned migration 001013.

CREATE TABLE IF NOT EXISTS organize_discover_categories (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'enabled',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CHECK (status IN ('enabled', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_organize_discover_categories_listing
    ON organize_discover_categories(status, sort_order, created_at);

INSERT OR IGNORE INTO organize_discover_categories (id, key, label, sort_order)
VALUES
    ('discover_category_admissions_growth', 'admissions_growth', '招生增长', 10),
    ('discover_category_parent_service', 'parent_service', '家长服务', 20),
    ('discover_category_event_planning', 'event_planning', '活动策划', 30),
    ('discover_category_kindergarten_operations', 'kindergarten_operations', '园所运营', 40),
    ('discover_category_team_leadership', 'team_leadership', '团队运营/领导力', 50),
    ('discover_category_nutrition_food_education', 'nutrition_food_education', '儿童营养与食育', 60),
    ('discover_category_space_environment', 'space_environment', '空间设计 / 环创', 70),
    ('discover_category_teacher_research', 'teacher_research', '教师成长 / 教研', 80);
