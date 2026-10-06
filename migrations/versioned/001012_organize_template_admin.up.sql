-- Migration 001012: platform organize-template governance metadata.

ALTER TABLE organize_templates
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS published_by VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS validation_result JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE organize_template_versions
    ADD COLUMN IF NOT EXISTS change_note TEXT NOT NULL DEFAULT '';

INSERT INTO organize_templates (
    id, tenant_id, owner_user_id, scope, key, name, scene, description,
    output_label, icon, default_instruction, expert_ids, spec, status,
    published_version, sort_order
) VALUES
    ('note_import_meta', 0, '', 'platform', 'note_import_meta', '文件导入元数据', '记忆导入',
     '为导入文件生成标题、摘要和标签', '笔记元数据', 'file-paste',
     '只输出 JSON，格式为 {"title":"标题","summary":"摘要","tags":["标签"]}。标题简短准确，摘要用 1-2 句话说明文件主要信息，标签使用具体中文词组，最多 {{max_tags}} 个，避免“文档”“文件”“其他”这类泛词。',
     '[]', '{"response_format":"json","fields":["title","summary","tags"]}', 'enabled', 'v1', 90),
    ('note_audio_transcribe', 0, '', 'platform', 'note_audio_transcribe', '录音笔记整理', '录音转写',
     '把录音转写整理成可阅读的 Markdown 笔记', '录音笔记', 'sound',
     '只输出 JSON，格式为 {"title":"标题","summary":"摘要","tags":["标签"],"note_markdown":"Markdown 笔记"}。笔记只整理转写内容，不得编造，优先呈现摘要、关键要点和明确的行动项，标签最多 {{max_tags}} 个。',
     '[]', '{"response_format":"json","fields":["title","summary","tags","note_markdown"]}', 'enabled', 'v1', 91),
    ('output_card_meta', 0, '', 'platform', 'output_card_meta', '成果卡片元数据', '成果分享',
     '为上传成果生成标题、摘要和标签', '成果卡片', 'file-paste',
     '只输出 JSON，格式为 {"title":"标题","summary":"摘要","tags":["标签"]}。标题、摘要和标签必须依据输入内容，摘要说明内容能帮助观看者提升什么业务能力，标签最多 {{max_tags}} 个，不得编造。',
     '[]', '{"response_format":"json","fields":["title","summary","tags"]}', 'enabled', 'v1', 92)
ON CONFLICT DO NOTHING;

INSERT INTO organize_template_versions (
    id, template_id, template_key, version, snapshot, created_by, change_note
)
SELECT
    key || ':v1',
    id,
    key,
    'v1',
    jsonb_build_object(
        'name', name,
        'scene', scene,
        'description', description,
        'output_label', output_label,
        'icon', icon,
        'default_instruction', default_instruction,
        'expert_ids', expert_ids,
        'spec', spec,
        'sort_order', sort_order
    ),
    'system',
    '系统内置模板'
FROM organize_templates
WHERE scope = 'platform'
  AND published_version = 'v1'
  AND key IN ('note_import_meta', 'note_audio_transcribe', 'output_card_meta')
ON CONFLICT DO NOTHING;
