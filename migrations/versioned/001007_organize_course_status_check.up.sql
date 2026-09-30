-- Migration 001007: allow the course review states on databases that already
-- applied 001005. Do not edit an applied migration to change this constraint.
ALTER TABLE organize_courses
    DROP CONSTRAINT IF EXISTS chk_organize_courses_public_status;

ALTER TABLE organize_courses
    ADD CONSTRAINT chk_organize_courses_public_status
    CHECK (public_status IN ('draft', 'pending_review', 'published', 'offline', 'rejected'));
