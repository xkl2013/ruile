-- Reverse of 001006: put the lesson bodies back into the public content pool.
--
-- This restores the publication flag only. published_at / published_by were
-- cleared by the up migration and cannot be reconstructed from any surviving
-- column, so published_at falls back to the row's last update time; the
-- publisher is left empty rather than attributed to the wrong actor.
UPDATE organize_outputs
   SET public_status = 'published',
       published_at  = COALESCE(published_at, updated_at)
 WHERE series_id IN (SELECT id FROM organize_courses);
