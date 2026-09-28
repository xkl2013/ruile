-- Migration 000114: backfill the default reminder status machine for legacy
-- service spaces created before migration 000113 was deployed.

INSERT INTO service_reminder_statuses (
    id, tenant_id, service_id, status_key, label, category,
    is_initial, is_terminal, display_order, is_system, enabled, created_by
)
SELECT
    uuid_generate_v4()::text, s.tenant_id, s.id, d.status_key, d.label, d.category,
    d.is_initial, d.is_terminal, d.display_order, true, true, s.created_by
FROM services s
CROSS JOIN (VALUES
    ('candidate', '待识别', 'open', true, false, 10),
    ('pending', '待处理', 'open', false, false, 20),
    ('generated', '已生成', 'in_progress', false, false, 30),
    ('confirmed', '已确认', 'in_progress', false, false, 40),
    ('completed', '已完成', 'done', false, true, 50),
    ('ignored', '已忽略', 'dismissed', false, true, 60),
    ('snoozed', '已延后', 'in_progress', false, false, 70),
    ('stale', '已过期', 'open', false, false, 80),
    ('recompute_required', '待重新计算', 'open', false, false, 90)
) AS d(status_key, label, category, is_initial, is_terminal, display_order)
WHERE s.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM service_reminder_statuses existing
      WHERE existing.service_id = s.id AND existing.status_key = d.status_key
  );

INSERT INTO service_reminder_status_transitions (
    id, tenant_id, service_id, from_status_id, to_status_id, allowed_roles, enabled
)
SELECT
    uuid_generate_v4()::text, from_status.tenant_id, from_status.service_id,
    from_status.id, to_status.id, '[]'::jsonb, true
FROM service_reminder_statuses from_status
JOIN service_reminder_statuses to_status
  ON to_status.service_id = from_status.service_id
JOIN (VALUES
    ('candidate', 'pending'),
    ('candidate', 'ignored'),
    ('pending', 'generated'),
    ('pending', 'ignored'),
    ('pending', 'snoozed'),
    ('generated', 'confirmed'),
    ('generated', 'ignored'),
    ('confirmed', 'completed'),
    ('confirmed', 'ignored'),
    ('snoozed', 'pending'),
    ('stale', 'pending'),
    ('recompute_required', 'pending')
) AS edge(from_key, to_key)
  ON edge.from_key = from_status.status_key AND edge.to_key = to_status.status_key
WHERE NOT EXISTS (
    SELECT 1 FROM service_reminder_status_transitions existing
    WHERE existing.service_id = from_status.service_id
      AND existing.from_status_id = from_status.id
      AND existing.to_status_id = to_status.id
);

