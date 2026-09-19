CREATE TABLE IF NOT EXISTS billing_service_prices (
    id VARCHAR(36) PRIMARY KEY,
    service_code VARCHAR(64) NOT NULL,
    service_name VARCHAR(128) NOT NULL DEFAULT '',
    pricing_mode VARCHAR(32) NOT NULL DEFAULT 'call'
        CHECK (pricing_mode IN ('call', 'unit')),
    nanousd_per_call BIGINT NOT NULL DEFAULT 0 CHECK (nanousd_per_call >= 0),
    nanousd_per_unit BIGINT NOT NULL DEFAULT 0 CHECK (nanousd_per_unit >= 0),
    unit_name VARCHAR(32) NOT NULL DEFAULT '',
    service_multiplier_ppm BIGINT NOT NULL DEFAULT 1000000
        CHECK (service_multiplier_ppm > 0),
    version INTEGER NOT NULL DEFAULT 1,
    effective_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (service_code, version)
);

CREATE INDEX IF NOT EXISTS idx_billing_service_prices_active
    ON billing_service_prices(service_code, status, effective_at DESC);

ALTER TABLE tenant_usage_reservations
    ADD COLUMN IF NOT EXISTS service_pricing_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE tenant_usage_ledgers
    ADD COLUMN IF NOT EXISTS service_pricing_id VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS service_pricing_version INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS service_units BIGINT NOT NULL DEFAULT 0;

INSERT INTO billing_service_prices (
    id,
    service_code,
    service_name,
    pricing_mode,
    nanousd_per_call,
    nanousd_per_unit,
    unit_name,
    service_multiplier_ppm,
    version,
    effective_at,
    status,
    snapshot_json
)
VALUES
    ('service-price-chat-completion-v1', 'chat.completion', '对话补充服务', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-agent-run-v1', 'agent.run', '智能体运行', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-embedding-v1', 'knowledge.embedding', '知识库向量化', 'unit', 0, 0, 'Token', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-summary-v1', 'knowledge.summary', '知识摘要', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-question-v1', 'knowledge.question_generation', '问题生成', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-graph-v1', 'knowledge.graph_extract', '知识图谱抽取', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-ocr-v1', 'file.ocr', 'OCR', 'unit', 0, 0, '页', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-asr-v1', 'file.asr', '语音识别', 'unit', 0, 0, '秒', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-rerank-v1', 'retrieval.rerank', '重排序', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-web-search-v1', 'web_search.query', '联网搜索', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-mcp-v1', 'mcp.tool_call', 'MCP 工具调用', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-embed-chat-v1', 'embed.chat', '嵌入式对话', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb),
    ('service-price-im-chat-v1', 'im.chat', '即时通讯对话', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'::jsonb)
ON CONFLICT (service_code, version) DO NOTHING;
