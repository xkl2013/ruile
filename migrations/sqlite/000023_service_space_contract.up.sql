-- Migration 000023: service-space V1-1 data contract.

ALTER TABLE services ADD COLUMN space_type TEXT NOT NULL DEFAULT 'customer_service';
ALTER TABLE services ADD COLUMN state_changed_at DATETIME;
ALTER TABLE services ADD COLUMN state_change_reason TEXT NOT NULL DEFAULT '';

UPDATE services
SET state_changed_at = COALESCE(updated_at, created_at, CURRENT_TIMESTAMP)
WHERE state_changed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_services_type_state
    ON services(tenant_id, space_type, state, updated_at DESC);

CREATE TRIGGER IF NOT EXISTS trg_services_space_type_insert
BEFORE INSERT ON services
WHEN NEW.space_type NOT IN ('customer_service', 'operations', 'research')
BEGIN
    SELECT RAISE(ABORT, 'invalid service space type');
END;

CREATE TRIGGER IF NOT EXISTS trg_services_space_type_update
BEFORE UPDATE OF space_type ON services
WHEN NEW.space_type NOT IN ('customer_service', 'operations', 'research')
BEGIN
    SELECT RAISE(ABORT, 'invalid service space type');
END;

CREATE TRIGGER IF NOT EXISTS trg_services_state_changed_at_insert
AFTER INSERT ON services
WHEN NEW.state_changed_at IS NULL
BEGIN
    UPDATE services
    SET state_changed_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;
