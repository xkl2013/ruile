-- Migration 000108: service-space V1-1 data contract.

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
