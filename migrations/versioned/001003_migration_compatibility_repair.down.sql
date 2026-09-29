-- The compatibility repair is additive and intentionally does not remove
-- objects that may have been created by either migration branch.

-- This repair reconciles schemas that may have come from either branch.
-- It must not drop objects owned by an earlier migration.
SELECT 1;
