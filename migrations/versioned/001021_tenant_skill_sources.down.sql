ALTER TABLE tenant_skills
    DROP COLUMN IF EXISTS source_ref;

ALTER TABLE tenant_skills
    DROP COLUMN IF EXISTS source_type;
