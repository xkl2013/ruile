DROP INDEX IF EXISTS idx_organize_job_batches_status;
DROP INDEX IF EXISTS idx_organize_job_batches_scope;
DROP INDEX IF EXISTS idx_organize_job_batches_parent_index;
DROP TABLE IF EXISTS organize_job_batches;

DROP INDEX IF EXISTS idx_organize_jobs_input_fingerprint;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS coverage;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS batch_count;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS overlap_count;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS failed_count;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS processed_count;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS ready_count;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS selected_count;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS input_fingerprint;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS selection_snapshot;
ALTER TABLE organize_jobs DROP COLUMN IF EXISTS job_mode;
