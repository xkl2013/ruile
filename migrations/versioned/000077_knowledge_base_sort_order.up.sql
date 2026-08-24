ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_tenant_sort
    ON knowledge_bases(tenant_id, sort_order, created_at);
