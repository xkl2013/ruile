CREATE TABLE IF NOT EXISTS billing_service_prices (
    id TEXT PRIMARY KEY,
    service_code TEXT NOT NULL,
    service_name TEXT NOT NULL DEFAULT '',
    pricing_mode TEXT NOT NULL DEFAULT 'call',
    nanousd_per_call INTEGER NOT NULL DEFAULT 0,
    nanousd_per_unit INTEGER NOT NULL DEFAULT 0,
    unit_name TEXT NOT NULL DEFAULT '',
    service_multiplier_ppm INTEGER NOT NULL DEFAULT 1000000,
    version INTEGER NOT NULL DEFAULT 1,
    effective_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    status TEXT NOT NULL DEFAULT 'active',
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (service_code, version)
);

CREATE INDEX IF NOT EXISTS idx_billing_service_prices_active
    ON billing_service_prices(service_code, status, effective_at DESC);

ALTER TABLE tenant_usage_reservations
    ADD COLUMN service_pricing_id TEXT NOT NULL DEFAULT '';

ALTER TABLE tenant_usage_ledgers
    ADD COLUMN service_pricing_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tenant_usage_ledgers
    ADD COLUMN service_pricing_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tenant_usage_ledgers
    ADD COLUMN service_units INTEGER NOT NULL DEFAULT 0;

INSERT OR IGNORE INTO billing_service_prices (
    id, service_code, service_name, pricing_mode, nanousd_per_call,
    nanousd_per_unit, unit_name, service_multiplier_ppm, version,
    effective_at, status, snapshot_json
)
VALUES
    ('service-price-chat-completion-v1', 'chat.completion', '对话补充服务', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-agent-run-v1', 'agent.run', '智能体运行', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-embedding-v1', 'knowledge.embedding', '知识库向量化', 'unit', 0, 0, 'Token', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-summary-v1', 'knowledge.summary', '知识摘要', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-question-v1', 'knowledge.question_generation', '问题生成', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-graph-v1', 'knowledge.graph_extract', '知识图谱抽取', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-ocr-v1', 'file.ocr', 'OCR', 'unit', 0, 0, '页', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-asr-v1', 'file.asr', '语音识别', 'unit', 0, 0, '秒', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-rerank-v1', 'retrieval.rerank', '重排序', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-web-search-v1', 'web_search.query', '联网搜索', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-mcp-v1', 'mcp.tool_call', 'MCP 工具调用', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-embed-chat-v1', 'embed.chat', '嵌入式对话', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}'),
    ('service-price-im-chat-v1', 'im.chat', '即时通讯对话', 'call', 0, 0, '次', 1000000, 1, CURRENT_TIMESTAMP, 'active', '{"bootstrap":true}');
