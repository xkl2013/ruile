ALTER TABLE organize_configs
    DROP COLUMN IF EXISTS schedule_time;
ALTER TABLE organize_templates
    DROP COLUMN IF EXISTS schedule_time;
