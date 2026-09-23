-- Migration 000106: durable organize templates, configs, jobs, and structured outputs.

CREATE TABLE IF NOT EXISTS organize_templates (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    owner_user_id VARCHAR(36) NOT NULL DEFAULT '',
    scope VARCHAR(32) NOT NULL DEFAULT 'platform',
    key VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    scene VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    output_label VARCHAR(128) NOT NULL DEFAULT '',
    icon VARCHAR(64) NOT NULL DEFAULT '',
    default_instruction TEXT NOT NULL DEFAULT '',
    expert_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    spec JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    published_version VARCHAR(32) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_templates_scope
        CHECK (scope IN ('platform', 'tenant', 'personal')),
    CONSTRAINT chk_organize_templates_status
        CHECK (status IN ('draft', 'enabled', 'disabled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_templates_scope_key
    ON organize_templates(tenant_id, owner_user_id, key)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_templates_visible
    ON organize_templates(status, scope, tenant_id, sort_order)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS organize_template_versions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    template_id VARCHAR(36) NOT NULL,
    template_key VARCHAR(64) NOT NULL,
    version VARCHAR(32) NOT NULL,
    snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_template_versions_unique
    ON organize_template_versions(template_id, version);
CREATE INDEX IF NOT EXISTS idx_organize_template_versions_key
    ON organize_template_versions(template_key, created_at DESC);

CREATE TABLE IF NOT EXISTS organize_configs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    template_key VARCHAR(64) NOT NULL,
    instruction TEXT NOT NULL DEFAULT '',
    expert_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    schedule VARCHAR(32) NOT NULL DEFAULT 'manual',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    next_run_at TIMESTAMP WITH TIME ZONE,
    last_run_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_configs_schedule
        CHECK (schedule IN ('manual', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_organize_configs_status
        CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_organize_configs_scope
    ON organize_configs(tenant_id, user_id, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_configs_due
    ON organize_configs(status, next_run_at)
    WHERE deleted_at IS NULL AND next_run_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS organize_jobs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    config_id VARCHAR(36) NOT NULL,
    template_key VARCHAR(64) NOT NULL,
    template_version VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    stage VARCHAR(64) NOT NULL DEFAULT 'queued',
    progress INTEGER NOT NULL DEFAULT 0,
    requirement JSONB NOT NULL DEFAULT '{}'::jsonb,
    memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    model_id VARCHAR(64) NOT NULL DEFAULT '',
    prompt_hash VARCHAR(64) NOT NULL DEFAULT '',
    output_id VARCHAR(36) NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    dedupe_key VARCHAR(255) NOT NULL DEFAULT '',
    scheduled_for TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_organize_jobs_status
        CHECK (status IN ('queued', 'running', 'repairing', 'completed', 'fallback', 'failed', 'canceled')),
    CONSTRAINT chk_organize_jobs_progress
        CHECK (progress >= 0 AND progress <= 100)
);

CREATE INDEX IF NOT EXISTS idx_organize_jobs_scope
    ON organize_jobs(tenant_id, user_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_jobs_config
    ON organize_jobs(config_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organize_jobs_status
    ON organize_jobs(status, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_organize_jobs_dedupe
    ON organize_jobs(tenant_id, user_id, dedupe_key)
    WHERE deleted_at IS NULL AND dedupe_key <> '';

ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS config_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS job_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS template_key VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS template_version VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS fields JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE organize_outputs ADD COLUMN IF NOT EXISTS citations JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_organize_outputs_config
    ON organize_outputs(tenant_id, user_id, config_id, updated_at DESC)
    WHERE deleted_at IS NULL AND config_id <> '';
CREATE INDEX IF NOT EXISTS idx_organize_outputs_job
    ON organize_outputs(job_id)
    WHERE deleted_at IS NULL AND job_id <> '';
CREATE INDEX IF NOT EXISTS idx_organize_outputs_template
    ON organize_outputs(template_key, updated_at DESC)
    WHERE deleted_at IS NULL AND template_key <> '';
CREATE INDEX IF NOT EXISTS idx_organize_outputs_fields_gin
    ON organize_outputs USING GIN(fields);

ALTER TABLE organize_sprout_reports ADD COLUMN IF NOT EXISTS template_key VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE organize_sprout_reports ADD COLUMN IF NOT EXISTS template_version VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE organize_sprout_reports ADD COLUMN IF NOT EXISTS fields JSONB NOT NULL DEFAULT '{}'::jsonb;

INSERT INTO organize_templates (
    id, tenant_id, owner_user_id, scope, key, name, scene, description,
    output_label, icon, default_instruction, expert_ids, spec, status,
    published_version, sort_order
) VALUES
    ('teacher_research', 0, '', 'platform', 'teacher_research', '教研提炼', '教师成长',
     '把教研记录整理成可复用的提炼清单', '提炼清单', 'chart-bar',
     '把选中的教研与磨课记录整理成提炼清单：先归纳儿童的真实问题，再给可复用的做法，最后列待办。每条结论标注来源记忆。',
     '[]'::jsonb, '{"tags":["教研","磨课"],"sections":["关键发现","可复用做法","待办"]}'::jsonb, 'enabled', 'v1', 10),
    ('lesson_review', 0, '', 'platform', 'lesson_review', '磨课复盘', '教师成长',
     '把一次磨课整理成复盘记录', '复盘记录', 'file-paste',
     '把一次磨课记录整理成复盘：环节回顾、儿童反应、改进点和下次尝试。只依据记录原文，不补充没有的内容。',
     '[]'::jsonb, '{"tags":["教研","复盘"],"sections":["环节回顾","儿童反应","改进点","下次尝试"]}'::jsonb, 'enabled', 'v1', 20),
    ('case_observation', 0, '', 'platform', 'case_observation', '个案观察', '教师成长',
     '把连续观察整理成个案跟踪记录', '观察记录', 'user',
     '把对同一名儿童的多次观察整理成个案跟踪记录，按时间轴呈现变化，并标注每个阶段的支持方式。',
     '[]'::jsonb, '{"tags":["观察","个案"],"sections":["观察时间轴","变化判断","支持方式"]}'::jsonb, 'enabled', 'v1', 30),
    ('lead_followup', 0, '', 'platform', 'lead_followup', '线索跟进清单', '招生增长',
     '把试听与咨询整理成待跟进清单', '跟进清单', 'usergroup',
     '把试听与咨询记录整理成跟进清单：按线索状态分组，标注流失风险与下一步动作，48 小时内的回访置顶。',
     '[]'::jsonb, '{"tags":["招生","跟进"],"sections":["线索分组","风险判断","下一步动作"]}'::jsonb, 'enabled', 'v1', 40),
    ('channel_report', 0, '', 'platform', 'channel_report', '渠道效果周报', '招生增长',
     '把各渠道记录整理成效果对比', '效果周报', 'chart',
     '把各渠道到访记录整理成效果周报，呈现渠道、到访数、转化数和单位成本。没有数据的字段明确标注缺失，不得编造。',
     '[]'::jsonb, '{"tags":["招生","渠道"],"sections":["渠道表现","数据缺口","下周动作"]}'::jsonb, 'enabled', 'v1', 50),
    ('parent_review', 0, '', 'platform', 'parent_review', '沟通复盘', '家长服务',
     '把家长沟通整理成问题与跟进清单', '沟通复盘', 'chat',
     '把家长沟通记录整理成复盘：问题类型、处理方式、当前状态和跟进时间，需要园所回应的内容排在最前。',
     '[]'::jsonb, '{"tags":["家长服务","沟通"],"sections":["问题类型","处理情况","待回应事项"]}'::jsonb, 'enabled', 'v1', 60),
    ('env_proposal', 0, '', 'platform', 'env_proposal', '环创提案', '空间设计',
     '把环创讨论整理成可执行的材料提案', '环创提案', 'layers',
     '把环创讨论整理成提案：现状、材料清单、预估成本和实施步骤，并附儿童使用动线的考虑。',
     '[]'::jsonb, '{"tags":["环创","空间"],"sections":["现状","材料与成本","实施步骤","儿童动线"]}'::jsonb, 'enabled', 'v1', 70),
    ('sprout_review', 0, '', 'platform', 'sprout_review', '发芽复盘', '个人沉淀',
     '整理记忆中的创意，提炼可继续发展的想法', '发芽复盘', 'tree-list',
     '整理记忆中的创意：保留原始想法和事实，提炼值得继续发展的方向，标注可以验证或展开的线索，并列出下一步尝试。',
     '[]'::jsonb, '{"tags":["发芽","复盘"],"sections":["原始种子","可发展方向","验证线索","下一步尝试"]}'::jsonb, 'enabled', 'v1', 80)
ON CONFLICT DO NOTHING;

INSERT INTO organize_template_versions (id, template_id, template_key, version, snapshot)
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
        'spec', spec
    )
FROM organize_templates
WHERE scope = 'platform' AND published_version = 'v1'
ON CONFLICT DO NOTHING;
