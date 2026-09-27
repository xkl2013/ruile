ALTER TABLE services DROP CONSTRAINT IF EXISTS chk_services_space_type;
DROP INDEX IF EXISTS idx_services_type_state;
ALTER TABLE services DROP COLUMN IF EXISTS state_change_reason;
ALTER TABLE services DROP COLUMN IF EXISTS state_changed_at;
ALTER TABLE services DROP COLUMN IF EXISTS space_type;
