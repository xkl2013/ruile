-- SQLite equivalent of 000098_dedupe_global_models.
-- Keep one active model per (name, type, source) after platform promotion.

CREATE TEMP TABLE model_dedup_redirects (
    seq INTEGER PRIMARY KEY,
    old_id VARCHAR(64) NOT NULL UNIQUE,
    new_id VARCHAR(64) NOT NULL
);

INSERT INTO model_dedup_redirects (seq, old_id, new_id)
SELECT
    ROW_NUMBER() OVER (ORDER BY duplicate.id),
    duplicate.id,
    (
        SELECT preferred.id
        FROM models preferred
        WHERE LOWER(TRIM(preferred.name)) = LOWER(TRIM(duplicate.name))
          AND preferred.type = duplicate.type
          AND preferred.source = duplicate.source
          AND preferred.deleted_at IS NULL
        ORDER BY
            CASE
                WHEN instr(
                    COALESCE((SELECT value FROM system_settings WHERE key = 'agent.response_tier_config'), ''),
                    preferred.id
                ) > 0 THEN 0 ELSE 1
            END,
            CASE
                WHEN COALESCE(json_extract(preferred.parameters, '$.api_key'), '') <> ''
                THEN 0 ELSE 1
            END,
            datetime(preferred.updated_at) DESC,
            datetime(preferred.created_at) DESC,
            preferred.id
        LIMIT 1
    )
FROM models duplicate
WHERE duplicate.deleted_at IS NULL
  AND TRIM(duplicate.name) <> ''
  AND duplicate.id <> (
      SELECT preferred.id
      FROM models preferred
      WHERE LOWER(TRIM(preferred.name)) = LOWER(TRIM(duplicate.name))
        AND preferred.type = duplicate.type
        AND preferred.source = duplicate.source
        AND preferred.deleted_at IS NULL
      ORDER BY
          CASE
              WHEN instr(
                  COALESCE((SELECT value FROM system_settings WHERE key = 'agent.response_tier_config'), ''),
                  preferred.id
              ) > 0 THEN 0 ELSE 1
          END,
          CASE
              WHEN COALESCE(json_extract(preferred.parameters, '$.api_key'), '') <> ''
              THEN 0 ELSE 1
          END,
          datetime(preferred.updated_at) DESC,
          datetime(preferred.created_at) DESC,
          preferred.id
      LIMIT 1
  );

UPDATE knowledge_bases
SET embedding_model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = knowledge_bases.embedding_model_id),
    embedding_model_id
),
summary_model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = knowledge_bases.summary_model_id),
    summary_model_id
);

UPDATE knowledges
SET embedding_model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = knowledges.embedding_model_id),
    embedding_model_id
);

UPDATE sessions
SET rerank_model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = sessions.rerank_model_id),
    rerank_model_id
),
summary_model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = sessions.summary_model_id),
    summary_model_id
);

UPDATE messages
SET model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = messages.model_id),
    model_id
);

UPDATE message_suggestion_sets
SET model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = message_suggestion_sets.model_id),
    model_id
);

UPDATE tenant_usage_ledgers
SET model_id = COALESCE(
    (SELECT new_id FROM model_dedup_redirects WHERE old_id = tenant_usage_ledgers.model_id),
    model_id
);

UPDATE models
SET is_default = 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (
    SELECT redirect.new_id
    FROM model_dedup_redirects redirect
    JOIN models duplicate ON duplicate.id = redirect.old_id
    WHERE duplicate.is_default = 1
);

-- SQLite stores the JSON columns as text. Fold every redirect through each
-- JSON-bearing column so all duplicate IDs are replaced, not just the first.
WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT kb.id, COALESCE(kb.image_processing_config, ''), 0
    FROM knowledge_bases kb
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(kb.image_processing_config, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE knowledge_bases
SET image_processing_config = (
    SELECT value FROM rewritten
    WHERE rewritten.id = knowledge_bases.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT kb.id, COALESCE(kb.vlm_config, ''), 0
    FROM knowledge_bases kb
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(kb.vlm_config, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE knowledge_bases
SET vlm_config = (
    SELECT value FROM rewritten
    WHERE rewritten.id = knowledge_bases.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT kb.id, COALESCE(kb.ocr_config, ''), 0
    FROM knowledge_bases kb
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(kb.ocr_config, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE knowledge_bases
SET ocr_config = (
    SELECT value FROM rewritten
    WHERE rewritten.id = knowledge_bases.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT kb.id, COALESCE(kb.asr_config, ''), 0
    FROM knowledge_bases kb
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(kb.asr_config, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE knowledge_bases
SET asr_config = (
    SELECT value FROM rewritten
    WHERE rewritten.id = knowledge_bases.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT kb.id, COALESCE(kb.wiki_config, ''), 0
    FROM knowledge_bases kb
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(kb.wiki_config, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE knowledge_bases
SET wiki_config = (
    SELECT value FROM rewritten
    WHERE rewritten.id = knowledge_bases.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT agent.id, COALESCE(agent.config, ''), 0
    FROM custom_agents agent
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(agent.config, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE custom_agents
SET config = (
    SELECT value FROM rewritten
    WHERE rewritten.id = custom_agents.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

WITH RECURSIVE rewritten(id, value, seq) AS (
    SELECT setting.id, COALESCE(setting.value, ''), 0
    FROM system_settings setting
    WHERE EXISTS (
        SELECT 1 FROM model_dedup_redirects r
        WHERE instr(COALESCE(setting.value, ''), r.old_id) > 0
    )
    UNION ALL
    SELECT rewritten.id, REPLACE(rewritten.value, r.old_id, r.new_id), r.seq
    FROM rewritten
    JOIN model_dedup_redirects r ON r.seq = rewritten.seq + 1
)
UPDATE system_settings
SET value = (
    SELECT value FROM rewritten
    WHERE rewritten.id = system_settings.id
    ORDER BY seq DESC
    LIMIT 1
)
WHERE id IN (SELECT id FROM rewritten);

UPDATE models
SET deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT old_id FROM model_dedup_redirects);

CREATE UNIQUE INDEX IF NOT EXISTS uq_models_platform_identity
    ON models (LOWER(TRIM(name)), type, source)
    WHERE deleted_at IS NULL AND TRIM(name) <> '';
