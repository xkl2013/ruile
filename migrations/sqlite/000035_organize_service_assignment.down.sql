DROP INDEX IF EXISTS idx_organize_outputs_assigned_service;
DROP INDEX IF EXISTS idx_organize_outputs_assignment;
DROP INDEX IF EXISTS idx_organize_jobs_target_service;
DROP INDEX IF EXISTS idx_organize_configs_target_service;

-- SQLite cannot safely drop columns on all supported versions. The columns
-- remain harmless when rolling back the feature migration.
