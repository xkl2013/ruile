DROP INDEX IF EXISTS idx_organize_outputs_assigned_service;
DROP INDEX IF EXISTS idx_organize_outputs_assignment;
DROP INDEX IF EXISTS idx_organize_jobs_target_service;
DROP INDEX IF EXISTS idx_organize_configs_target_service;

-- Keep the added columns during rollback for compatibility with SQLite and
-- older deployments that may still read the extended API shape.
