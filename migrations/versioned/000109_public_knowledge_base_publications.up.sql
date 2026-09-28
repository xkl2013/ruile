CREATE TABLE IF NOT EXISTS knowledge_base_publications (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    knowledge_base_id VARCHAR(36) NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    title VARCHAR(120) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    published_at TIMESTAMP WITH TIME ZONE,
    offline_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    updated_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_knowledge_base_publications_kb
    ON knowledge_base_publications(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_base_publications_status
    ON knowledge_base_publications(status, featured, sort_order, updated_at DESC);

CREATE TABLE IF NOT EXISTS public_knowledge_base_subscriptions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL,
    publication_id VARCHAR(36) NOT NULL REFERENCES knowledge_base_publications(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cancelled_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_public_kb_subscriptions_user_publication
    ON public_knowledge_base_subscriptions(user_id, publication_id);
CREATE INDEX IF NOT EXISTS idx_public_kb_subscriptions_user
    ON public_knowledge_base_subscriptions(user_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_public_kb_subscriptions_publication
    ON public_knowledge_base_subscriptions(publication_id, status);

COMMENT ON TABLE knowledge_base_publications IS
    'Platform-public knowledge-base publication metadata.';
COMMENT ON TABLE public_knowledge_base_subscriptions IS
    'Access-granting subscriptions to published platform knowledge bases.';
