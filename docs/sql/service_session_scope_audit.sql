-- Service session scope audit.
-- Read-only: this file never guesses ownership from title, timestamps, or user ID.

-- 1. Current distribution.
SELECT
    CASE WHEN COALESCE(service_id, '') = '' THEN 'global_or_unbound' ELSE 'service_bound' END AS scope,
    COUNT(*) AS session_count
FROM sessions
WHERE deleted_at IS NULL
GROUP BY 1
ORDER BY 1;

-- 2. Unbound sessions with exactly one provable service from their agent runs.
WITH run_scope AS (
    SELECT
        thread_id AS session_id,
        MIN(service_id) AS service_id,
        COUNT(DISTINCT service_id) AS service_count
    FROM agent_runs
    WHERE COALESCE(thread_id, '') <> ''
      AND COALESCE(service_id, '') <> ''
    GROUP BY thread_id
)
SELECT
    s.id AS session_id,
    s.tenant_id,
    s.user_id,
    r.service_id AS provable_service_id
FROM sessions s
JOIN run_scope r ON r.session_id = s.id AND r.service_count = 1
WHERE s.deleted_at IS NULL
  AND COALESCE(s.service_id, '') = ''
ORDER BY s.tenant_id, s.created_at, s.id;

-- 3. Ambiguous rows must remain untouched and be reviewed manually.
WITH run_scope AS (
    SELECT
        thread_id AS session_id,
        COUNT(DISTINCT service_id) AS service_count,
        STRING_AGG(DISTINCT service_id, ',' ORDER BY service_id) AS service_ids
    FROM agent_runs
    WHERE COALESCE(thread_id, '') <> ''
      AND COALESCE(service_id, '') <> ''
    GROUP BY thread_id
)
SELECT
    s.id AS session_id,
    s.tenant_id,
    s.user_id,
    r.service_count,
    r.service_ids
FROM sessions s
JOIN run_scope r ON r.session_id = s.id AND r.service_count > 1
WHERE s.deleted_at IS NULL
  AND COALESCE(s.service_id, '') = ''
ORDER BY s.tenant_id, s.created_at, s.id;

-- Any repair must use only rows returned by query 2 and must be executed as a
-- separately reviewed transaction with an expected row count.
