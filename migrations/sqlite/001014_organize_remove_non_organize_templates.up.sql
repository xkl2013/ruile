-- SQLite equivalent of versioned migration 001014.

DELETE FROM organize_template_versions
WHERE template_key IN ('note_import_meta', 'note_audio_transcribe', 'output_card_meta');

DELETE FROM organize_templates
WHERE tenant_id = 0
  AND owner_user_id = ''
  AND scope = 'platform'
  AND key IN ('note_import_meta', 'note_audio_transcribe', 'output_card_meta');
