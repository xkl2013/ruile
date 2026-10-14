DROP INDEX IF EXISTS idx_service_fact_proposals_assistant_message;

ALTER TABLE service_fact_proposals
    DROP COLUMN IF EXISTS assistant_message_id;
