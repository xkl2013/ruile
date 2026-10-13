-- Allow imported expert Agents to retain protected resource references and URLs.
ALTER TABLE custom_agents
    ALTER COLUMN avatar TYPE TEXT;
