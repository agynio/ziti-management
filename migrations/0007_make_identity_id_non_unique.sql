DROP INDEX IF EXISTS idx_managed_identities_identity_id;
CREATE INDEX IF NOT EXISTS idx_managed_identities_identity_id ON managed_identities (identity_id);
