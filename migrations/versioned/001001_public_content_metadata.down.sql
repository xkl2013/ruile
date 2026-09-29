-- Compatibility migration moved out of the conflicting 000110 branch range.
DROP INDEX IF EXISTS idx_organize_outputs_public_series;
DROP INDEX IF EXISTS idx_organize_outputs_public_status;

ALTER TABLE organize_outputs DROP COLUMN IF EXISTS published_by;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS published_at;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS review_note;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS series_order;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS series_title;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS series_id;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS public_status;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS public_content_type;
