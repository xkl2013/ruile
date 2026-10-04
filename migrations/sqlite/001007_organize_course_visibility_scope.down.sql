-- Reverse of 001007.
DROP INDEX IF EXISTS idx_organize_course_shared_spaces_org;
DROP INDEX IF EXISTS idx_organize_course_shared_spaces_unique;
DROP TABLE IF EXISTS organize_course_shared_spaces;
DROP INDEX IF EXISTS idx_organize_courses_visibility;
ALTER TABLE organize_courses DROP COLUMN visibility_scope;
