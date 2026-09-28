-- Reconcile migration branches that previously reused versions 000108 and
-- 000109. This migration is intentionally additive and idempotent so it can
-- repair databases that already applied either side of the branch merge.

CREATE TABLE IF NOT EXISTS tenant_skills (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    name VARCHAR(64) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version VARCHAR(64) NOT NULL DEFAULT '',
    bundle_path TEXT NOT NULL,
    bundle_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'ready',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_skills_name
    ON tenant_skills(tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_skills_tenant
    ON tenant_skills(tenant_id, enabled, updated_at DESC)
    WHERE deleted_at IS NULL;

ALTER TABLE services
    ADD COLUMN IF NOT EXISTS space_type VARCHAR(32) NOT NULL DEFAULT 'customer_service';
ALTER TABLE services
    ADD COLUMN IF NOT EXISTS state_changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE services
    ADD COLUMN IF NOT EXISTS state_change_reason TEXT NOT NULL DEFAULT '';

UPDATE services
SET state_changed_at = COALESCE(updated_at, created_at, CURRENT_TIMESTAMP)
WHERE state_changed_at IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_services_space_type'
    ) THEN
        ALTER TABLE services
            ADD CONSTRAINT chk_services_space_type
            CHECK (space_type IN ('customer_service', 'operations', 'research'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_services_type_state
    ON services(tenant_id, space_type, state, updated_at DESC)
    WHERE deleted_at IS NULL;

ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS service_id VARCHAR(36);
ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS subject_type VARCHAR(64);
ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS parent_subject_id VARCHAR(36);
ALTER TABLE service_subjects
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

DROP INDEX IF EXISTS idx_service_subjects_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_service_subjects_type_when_scoped'
    ) THEN
        ALTER TABLE service_subjects
            ADD CONSTRAINT chk_service_subjects_type_when_scoped
            CHECK (service_id IS NULL OR length(trim(COALESCE(subject_type, ''))) > 0)
            NOT VALID;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_subjects_service
    ON service_subjects(tenant_id, service_id, subject_type, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_subjects_parent
    ON service_subjects(tenant_id, service_id, parent_subject_id)
    WHERE parent_subject_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_subjects_service_key
    ON service_subjects(tenant_id, service_id, subject_key)
    WHERE service_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_subjects_legacy_key
    ON service_subjects(tenant_id, owner_user_id, subject_key)
    WHERE service_id IS NULL AND deleted_at IS NULL;

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
