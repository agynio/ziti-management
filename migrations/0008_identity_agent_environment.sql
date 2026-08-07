-- Workload identities record what they are configured to run, so a data-plane
-- service can answer it from the connection instead of asking Agents or
-- trusting the workload. Null on every other identity type, and agent_id is
-- null for a sandbox, which runs an environment with no agent behind it.
ALTER TABLE managed_identities ADD COLUMN agent_id UUID;
ALTER TABLE managed_identities ADD COLUMN environment_id UUID;
