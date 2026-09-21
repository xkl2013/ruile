DROP INDEX IF EXISTS uq_models_platform_identity;

-- The original per-workspace duplicate rows cannot be reconstructed after
-- references have been redirected and redundant rows soft-deleted.
