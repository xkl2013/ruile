ALTER TABLE organize_template_versions
    DROP COLUMN IF EXISTS change_note;

ALTER TABLE organize_templates
    DROP COLUMN IF EXISTS validation_result,
    DROP COLUMN IF EXISTS published_by,
    DROP COLUMN IF EXISTS published_at;
