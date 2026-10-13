ALTER TABLE custom_agents
    ALTER COLUMN avatar TYPE VARCHAR(64)
    USING LEFT(avatar, 64);
