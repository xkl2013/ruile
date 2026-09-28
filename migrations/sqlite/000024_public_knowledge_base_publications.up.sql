CREATE TABLE IF NOT EXISTS knowledge_base_publications (
    id TEXT PRIMARY KEY,
    knowledge_base_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    featured INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    published_at DATETIME,
    offline_at DATETIME,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_knowledge_base_publications_kb
    ON knowledge_base_publications(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_base_publications_status
    ON knowledge_base_publications(status, featured, sort_order, updated_at DESC);

CREATE TABLE IF NOT EXISTS public_knowledge_base_subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    publication_id TEXT NOT NULL REFERENCES knowledge_base_publications(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cancelled_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_public_kb_subscriptions_user_publication
    ON public_knowledge_base_subscriptions(user_id, publication_id);
CREATE INDEX IF NOT EXISTS idx_public_kb_subscriptions_user
    ON public_knowledge_base_subscriptions(user_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_public_kb_subscriptions_publication
    ON public_knowledge_base_subscriptions(publication_id, status);
