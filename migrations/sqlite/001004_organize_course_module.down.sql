-- Migration 001004 down: drop the organize course module tables.
DROP INDEX IF EXISTS idx_organize_course_lessons_course;
DROP INDEX IF EXISTS idx_organize_course_lessons_output;
DROP INDEX IF EXISTS idx_organize_course_lessons_order;
DROP TABLE IF EXISTS organize_course_lessons;

DROP INDEX IF EXISTS idx_organize_courses_scope_updated;
DROP INDEX IF EXISTS idx_organize_courses_discover;
DROP TABLE IF EXISTS organize_courses;
