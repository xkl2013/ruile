-- Migration 000033: backfill the default reminder status machine for legacy
-- service spaces created before migration 000032 was deployed.

INSERT INTO service_reminder_statuses (
    id, tenant_id, service_id, status_key, label, category,
    is_initial, is_terminal, display_order, is_system, enabled, created_by
)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' ||
        substr(lower(hex(randomblob(2))), 2) || '-' ||
        substr('89ab', abs(random()) % 4 + 1, 1) ||
        substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    s.tenant_id, s.id, d.status_key, d.label, d.category,
    d.is_initial, d.is_terminal, d.display_order, 1, 1, s.created_by
FROM services s
CROSS JOIN (
    SELECT 'candidate' AS status_key, '待识别' AS label, 'open' AS category, 1 AS is_initial, 0 AS is_terminal, 10 AS display_order
    UNION ALL SELECT 'pending', '待处理', 'open', 0, 0, 20
    UNION ALL SELECT 'generated', '已生成', 'in_progress', 0, 0, 30
    UNION ALL SELECT 'confirmed', '已确认', 'in_progress', 0, 0, 40
    UNION ALL SELECT 'completed', '已完成', 'done', 0, 1, 50
    UNION ALL SELECT 'ignored', '已忽略', 'dismissed', 0, 1, 60
    UNION ALL SELECT 'snoozed', '已延后', 'in_progress', 0, 0, 70
    UNION ALL SELECT 'stale', '已过期', 'open', 0, 0, 80
    UNION ALL SELECT 'recompute_required', '待重新计算', 'open', 0, 0, 90
) d
WHERE s.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM service_reminder_statuses existing
      WHERE existing.service_id = s.id AND existing.status_key = d.status_key
  );

INSERT INTO service_reminder_status_transitions (
    id, tenant_id, service_id, from_status_id, to_status_id, allowed_roles, enabled
)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' ||
        substr(lower(hex(randomblob(2))), 2) || '-' ||
        substr('89ab', abs(random()) % 4 + 1, 1) ||
        substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    from_status.tenant_id, from_status.service_id, from_status.id, to_status.id, '[]', 1
FROM service_reminder_statuses from_status
JOIN service_reminder_statuses to_status
  ON to_status.service_id = from_status.service_id
JOIN (
    SELECT 'candidate' AS from_key, 'pending' AS to_key
    UNION ALL SELECT 'candidate', 'ignored'
    UNION ALL SELECT 'pending', 'generated'
    UNION ALL SELECT 'pending', 'ignored'
    UNION ALL SELECT 'pending', 'snoozed'
    UNION ALL SELECT 'generated', 'confirmed'
    UNION ALL SELECT 'generated', 'ignored'
    UNION ALL SELECT 'confirmed', 'completed'
    UNION ALL SELECT 'confirmed', 'ignored'
    UNION ALL SELECT 'snoozed', 'pending'
    UNION ALL SELECT 'stale', 'pending'
    UNION ALL SELECT 'recompute_required', 'pending'
) edge
  ON edge.from_key = from_status.status_key AND edge.to_key = to_status.status_key
WHERE NOT EXISTS (
    SELECT 1 FROM service_reminder_status_transitions existing
    WHERE existing.service_id = from_status.service_id
      AND existing.from_status_id = from_status.id
      AND existing.to_status_id = to_status.id
);

