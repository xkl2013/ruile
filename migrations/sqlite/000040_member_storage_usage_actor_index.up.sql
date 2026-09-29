CREATE INDEX IF NOT EXISTS idx_tenant_storage_transactions_tenant_actor
    ON tenant_storage_transactions (tenant_id, actor_user_id);
