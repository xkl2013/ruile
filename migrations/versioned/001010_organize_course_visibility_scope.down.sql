-- Reverse of 001010.
DROP INDEX IF EXISTS idx_organize_course_shared_spaces_org;
DROP INDEX IF EXISTS idx_organize_course_shared_spaces_unique;
DROP TABLE IF EXISTS organize_course_shared_spaces;

DROP INDEX IF EXISTS idx_organize_courses_visibility;
ALTER TABLE organize_courses
    DROP CONSTRAINT IF EXISTS chk_organize_courses_visibility_scope;
ALTER TABLE organize_courses
    DROP COLUMN IF EXISTS visibility_scope;
