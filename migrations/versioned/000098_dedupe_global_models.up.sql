-- Deduplicate platform models after workspace-to-platform promotion.
--
-- Before 000096, each workspace could have a different UUID for the same
-- user-facing model. 000096 moved every row to tenant_id=0 but did not merge
-- those logical duplicates. Keep one row per (name, type, source), redirect
-- all known references, then soft-delete the redundant rows.

CREATE TEMP TABLE model_dedup_redirects (
    old_id VARCHAR(64) PRIMARY KEY,
    new_id VARCHAR(64) NOT NULL
) ON COMMIT DROP;

WITH ranked AS (
    SELECT
        m.id,
        FIRST_VALUE(m.id) OVER (
            PARTITION BY LOWER(BTRIM(m.name)), m.type, m.source
            ORDER BY
                CASE
                    WHEN EXISTS (
                        SELECT 1
                        FROM system_settings ss
                        WHERE ss.key = 'agent.response_tier_config'
                          AND ss.value::text LIKE '%' || m.id || '%'
                    ) THEN 0 ELSE 1
                END,
                CASE
                    WHEN COALESCE(m.parameters->>'api_key', '') <> '' THEN 0 ELSE 1
                END,
                m.updated_at DESC NULLS LAST,
                m.created_at DESC NULLS LAST,
                m.id
        ) AS canonical_id
    FROM models m
    WHERE m.deleted_at IS NULL
      AND BTRIM(m.name) <> ''
)
INSERT INTO model_dedup_redirects (old_id, new_id)
SELECT id, canonical_id
FROM ranked
WHERE id <> canonical_id;

UPDATE knowledge_bases kb
SET embedding_model_id = r.new_id
FROM model_dedup_redirects r
WHERE kb.embedding_model_id = r.old_id;

UPDATE knowledge_bases kb
SET summary_model_id = r.new_id
FROM model_dedup_redirects r
WHERE kb.summary_model_id = r.old_id;

UPDATE knowledges k
SET embedding_model_id = r.new_id
FROM model_dedup_redirects r
WHERE k.embedding_model_id = r.old_id;

UPDATE sessions s
SET rerank_model_id = r.new_id
FROM model_dedup_redirects r
WHERE s.rerank_model_id = r.old_id;

UPDATE sessions s
SET summary_model_id = r.new_id
FROM model_dedup_redirects r
WHERE s.summary_model_id = r.old_id;

UPDATE messages m
SET model_id = r.new_id
FROM model_dedup_redirects r
WHERE m.model_id = r.old_id;

UPDATE message_suggestion_sets s
SET model_id = r.new_id
FROM model_dedup_redirects r
WHERE s.model_id = r.old_id;

UPDATE tenant_usage_ledgers l
SET model_id = r.new_id
FROM model_dedup_redirects r
WHERE l.model_id = r.old_id;

-- Preserve a default flag if it existed on a row that is being merged.
UPDATE models canonical
SET is_default = TRUE,
    updated_at = CURRENT_TIMESTAMP
FROM (
    SELECT DISTINCT r.new_id
    FROM model_dedup_redirects r
    JOIN models duplicate ON duplicate.id = r.old_id
    WHERE duplicate.is_default = TRUE
) defaults
WHERE canonical.id = defaults.new_id;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT old_id, new_id FROM model_dedup_redirects LOOP
        UPDATE knowledge_bases
        SET image_processing_config = CASE
                WHEN image_processing_config->>'model_id' = r.old_id
                THEN jsonb_set(COALESCE(image_processing_config, '{}'::jsonb), '{model_id}', to_jsonb(r.new_id), TRUE)
                ELSE image_processing_config
            END,
            vlm_config = CASE
                WHEN vlm_config->>'model_id' = r.old_id
                THEN jsonb_set(COALESCE(vlm_config, '{}'::jsonb), '{model_id}', to_jsonb(r.new_id), TRUE)
                ELSE vlm_config
            END,
            ocr_config = CASE
                WHEN ocr_config->>'model_id' = r.old_id
                THEN jsonb_set(COALESCE(ocr_config, '{}'::jsonb), '{model_id}', to_jsonb(r.new_id), TRUE)
                ELSE ocr_config
            END,
            asr_config = CASE
                WHEN asr_config->>'model_id' = r.old_id
                THEN jsonb_set(COALESCE(asr_config, '{}'::jsonb), '{model_id}', to_jsonb(r.new_id), TRUE)
                ELSE asr_config
            END,
            wiki_config = CASE
                WHEN wiki_config->>'synthesis_model_id' = r.old_id
                THEN jsonb_set(COALESCE(wiki_config, '{}'::jsonb), '{synthesis_model_id}', to_jsonb(r.new_id), TRUE)
                ELSE wiki_config
            END
        WHERE image_processing_config->>'model_id' = r.old_id
           OR vlm_config->>'model_id' = r.old_id
           OR ocr_config->>'model_id' = r.old_id
           OR asr_config->>'model_id' = r.old_id
           OR wiki_config->>'synthesis_model_id' = r.old_id;

        UPDATE custom_agents
        SET config = jsonb_set(config, '{model_id}', to_jsonb(r.new_id), TRUE)
        WHERE config->>'model_id' = r.old_id;

        UPDATE custom_agents
        SET config = jsonb_set(config, '{rerank_model_id}', to_jsonb(r.new_id), TRUE)
        WHERE config->>'rerank_model_id' = r.old_id;

        UPDATE custom_agents
        SET config = jsonb_set(config, '{vlm_model_id}', to_jsonb(r.new_id), TRUE)
        WHERE config->>'vlm_model_id' = r.old_id;

        UPDATE custom_agents
        SET config = jsonb_set(config, '{asr_model_id}', to_jsonb(r.new_id), TRUE)
        WHERE config->>'asr_model_id' = r.old_id;

        UPDATE custom_agents
        SET config = jsonb_set(config, '{query_understand_model_id}', to_jsonb(r.new_id), TRUE)
        WHERE config->>'query_understand_model_id' = r.old_id;

        UPDATE custom_agents
        SET config = jsonb_set(
            config,
            '{question_suggestions,follow_ups,model_id}',
            to_jsonb(r.new_id),
            TRUE
        )
        WHERE config->'question_suggestions'->'follow_ups'->>'model_id' = r.old_id;

        UPDATE system_settings
        SET value = REPLACE(value::text, r.old_id, r.new_id)::jsonb,
            updated_at = CURRENT_TIMESTAMP
        WHERE value::text LIKE '%' || r.old_id || '%';
    END LOOP;
END $$;

UPDATE models m
SET deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
FROM model_dedup_redirects r
WHERE m.id = r.old_id
  AND m.deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_models_platform_identity
    ON models (LOWER(BTRIM(name)), type, source)
    WHERE deleted_at IS NULL AND BTRIM(name) <> '';
