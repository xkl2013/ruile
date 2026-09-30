-- Restore the original course status vocabulary. Existing review states are
-- normalized to draft before the narrower constraint is installed.
UPDATE organize_courses
   SET public_status = 'draft'
 WHERE public_status IN ('pending_review', 'rejected');

ALTER TABLE organize_courses
    DROP CONSTRAINT IF EXISTS chk_organize_courses_public_status;

ALTER TABLE organize_courses
    ADD CONSTRAINT chk_organize_courses_public_status
    CHECK (public_status IN ('draft', 'published', 'offline'));
