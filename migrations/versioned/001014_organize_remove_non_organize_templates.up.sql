-- Migration 001014: remove attachment/upload recipes from organize-template governance.
--
-- These recipes belong to memory attachment parsing and result-card upload
-- processing. They must not be exposed as organize templates or configurable
-- organize jobs.

DELETE FROM organize_template_versions
WHERE template_key IN ('note_import_meta', 'note_audio_transcribe', 'output_card_meta');

DELETE FROM organize_templates
WHERE tenant_id = 0
  AND owner_user_id = ''
  AND scope = 'platform'
  AND key IN ('note_import_meta', 'note_audio_transcribe', 'output_card_meta');
