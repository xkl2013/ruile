-- Migration 001002: public content metadata.
-- This was previously numbered 000025 on a parallel branch.

ALTER TABLE organize_outputs ADD COLUMN public_content_type VARCHAR(32) NOT NULL DEFAULT 'post';
ALTER TABLE organize_outputs ADD COLUMN public_status VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN series_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN series_title VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN series_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organize_outputs ADD COLUMN review_note TEXT NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN published_at DATETIME;
ALTER TABLE organize_outputs ADD COLUMN published_by VARCHAR(36) NOT NULL DEFAULT '';

UPDATE organize_outputs
SET public_status = CASE status
    WHEN 'ready' THEN 'published'
    WHEN 'review' THEN 'pending_review'
    ELSE ''
END
WHERE public_status = '';

CREATE INDEX IF NOT EXISTS idx_organize_outputs_public_status
    ON organize_outputs(public_status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_organize_outputs_public_series
    ON organize_outputs(series_id, series_order, updated_at DESC);
