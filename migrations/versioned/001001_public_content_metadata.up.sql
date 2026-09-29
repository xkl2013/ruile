-- Migration 001001: public content metadata.
-- This was previously numbered 000110 on a parallel branch. Keep the SQL
-- idempotent so databases that already applied that branch can be upgraded.

-- Public content metadata is kept on organize_outputs so existing upload,
-- parsing, preview, and editor flows remain the source of truth for media.
ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS public_content_type VARCHAR(32) NOT NULL DEFAULT 'post';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS public_status VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS series_id VARCHAR(128) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS series_title VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS series_order INTEGER NOT NULL DEFAULT 0;

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '';

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE organize_outputs
    ADD COLUMN IF NOT EXISTS published_by VARCHAR(36) NOT NULL DEFAULT '';

UPDATE organize_outputs
SET public_status = CASE status
    WHEN 'ready' THEN 'published'
    WHEN 'review' THEN 'pending_review'
    ELSE ''
END
WHERE public_status = '';

CREATE INDEX IF NOT EXISTS idx_organize_outputs_public_status
    ON organize_outputs(public_status, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_organize_outputs_public_series
    ON organize_outputs(series_id, series_order, updated_at DESC)
    WHERE deleted_at IS NULL;
