-- R0 compatibility check for PostgreSQL.
-- Run before and after an organize migration and compare the result rows.
-- This script is read-only and intentionally does not mutate production data.

SELECT 'organize_templates' AS table_name, COUNT(*) AS row_count
FROM organize_templates
UNION ALL
SELECT 'organize_template_versions', COUNT(*)
FROM organize_template_versions
UNION ALL
SELECT 'organize_configs', COUNT(*)
FROM organize_configs
UNION ALL
SELECT 'organize_jobs', COUNT(*)
FROM organize_jobs
UNION ALL
SELECT 'organize_outputs', COUNT(*)
FROM organize_outputs;

SELECT
  COUNT(*) FILTER (WHERE template_key <> '' AND job_id <> '') AS linked_outputs,
  COUNT(*) FILTER (WHERE template_key = '' OR job_id = '') AS historical_or_unlinked_outputs,
  COUNT(*) FILTER (WHERE output_id <> '' AND status IN ('completed', 'fallback')) AS terminal_jobs_with_output,
  COUNT(*) FILTER (WHERE output_id = '' AND status IN ('completed', 'fallback')) AS terminal_jobs_without_output
FROM organize_jobs;

SELECT
  status,
  COUNT(*) AS job_count
FROM organize_jobs
GROUP BY status
ORDER BY status;
