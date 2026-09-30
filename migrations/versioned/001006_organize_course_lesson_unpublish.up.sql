-- Lessons are not posts.
--
-- Course lessons were originally written into organize_outputs with the
-- course's own public_status, which put every lesson into the public content
-- pool and therefore into the discover feed as an independent card sitting next
-- to the course that already contains it.
--
-- A lesson body is reachable only through its course (GET
-- /organize/courses/:id, which gates on the course's public_status), so it must
-- never carry a public status of its own. Pull the existing rows back out.
UPDATE organize_outputs
   SET public_status = 'draft',
       published_at  = NULL,
       published_by  = ''
 WHERE series_id IN (SELECT id FROM organize_courses);
