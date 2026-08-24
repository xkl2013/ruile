DROP INDEX IF EXISTS idx_knowledge_bases_tenant_sort;

ALTER TABLE knowledge_bases
    DROP COLUMN IF EXISTS sort_order;
