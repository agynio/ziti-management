DROP INDEX IF EXISTS idx_managed_identities_tenant;
DROP INDEX IF EXISTS idx_managed_identities_type_tenant;

ALTER TABLE managed_identities DROP COLUMN tenant_id;

CREATE INDEX IF NOT EXISTS idx_managed_identities_type ON managed_identities (identity_type);
