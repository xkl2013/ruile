DROP INDEX IF EXISTS idx_organize_outputs_fields_gin;
DROP INDEX IF EXISTS idx_organize_outputs_template;
DROP INDEX IF EXISTS idx_organize_outputs_job;
DROP INDEX IF EXISTS idx_organize_outputs_config;

ALTER TABLE organize_sprout_reports DROP COLUMN IF EXISTS fields;
ALTER TABLE organize_sprout_reports DROP COLUMN IF EXISTS template_version;
ALTER TABLE organize_sprout_reports DROP COLUMN IF EXISTS template_key;

ALTER TABLE organize_outputs DROP COLUMN IF EXISTS citations;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS fields;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS template_version;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS template_key;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS job_id;
ALTER TABLE organize_outputs DROP COLUMN IF EXISTS config_id;

DROP TABLE IF EXISTS organize_jobs;
DROP TABLE IF EXISTS organize_configs;
DROP TABLE IF EXISTS organize_template_versions;
DROP TABLE IF EXISTS organize_templates;
