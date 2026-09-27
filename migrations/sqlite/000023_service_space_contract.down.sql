DROP TRIGGER IF EXISTS trg_services_space_type_update;
DROP TRIGGER IF EXISTS trg_services_space_type_insert;
DROP TRIGGER IF EXISTS trg_services_state_changed_at_insert;
DROP INDEX IF EXISTS idx_services_type_state;
ALTER TABLE services DROP COLUMN state_change_reason;
ALTER TABLE services DROP COLUMN state_changed_at;
ALTER TABLE services DROP COLUMN space_type;
