-- Migration 001019: store the default clock time for organize schedules.

ALTER TABLE organize_templates
    ADD COLUMN IF NOT EXISTS schedule_time VARCHAR(5) NOT NULL DEFAULT '';
ALTER TABLE organize_configs
    ADD COLUMN IF NOT EXISTS schedule_time VARCHAR(5) NOT NULL DEFAULT '';

UPDATE organize_templates
SET schedule_time = CASE schedule
    WHEN 'daily' THEN '08:00'
    WHEN 'weekly' THEN '09:00'
    WHEN 'monthly' THEN '09:00'
    ELSE ''
END
WHERE schedule_time = '';

UPDATE organize_configs
SET schedule_time = CASE schedule
    WHEN 'daily' THEN '08:00'
    WHEN 'weekly' THEN '09:00'
    WHEN 'monthly' THEN '09:00'
    ELSE ''
END
WHERE schedule_time = '';

-- "Yesterday's visit interviews" is a daily end-of-day review. Keep this deployment
-- template aligned with the instruction semantics.
UPDATE organize_templates
SET schedule = 'daily',
    schedule_time = '20:00'
WHERE tenant_id = 0
  AND scope = 'platform'
  AND key = 'visit_review';
