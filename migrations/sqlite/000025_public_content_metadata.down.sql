DROP INDEX IF EXISTS idx_organize_outputs_public_series;
DROP INDEX IF EXISTS idx_organize_outputs_public_status;

ALTER TABLE organize_outputs DROP COLUMN published_by;
ALTER TABLE organize_outputs DROP COLUMN published_at;
ALTER TABLE organize_outputs DROP COLUMN review_note;
ALTER TABLE organize_outputs DROP COLUMN series_order;
ALTER TABLE organize_outputs DROP COLUMN series_title;
ALTER TABLE organize_outputs DROP COLUMN series_id;
ALTER TABLE organize_outputs DROP COLUMN public_status;
ALTER TABLE organize_outputs DROP COLUMN public_content_type;
