-- Migration 001029: persist the assistant message that owns a service fact proposal.

ALTER TABLE service_fact_proposals
    ADD COLUMN IF NOT EXISTS assistant_message_id VARCHAR(36);

CREATE INDEX IF NOT EXISTS idx_service_fact_proposals_assistant_message
    ON service_fact_proposals(tenant_id, service_id, assistant_message_id)
    WHERE assistant_message_id IS NOT NULL AND assistant_message_id <> '';
