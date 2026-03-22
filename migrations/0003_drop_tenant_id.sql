DROP INDEX IF EXISTS idx_managed_identities_tenant;
DROP INDEX IF EXISTS idx_managed_identities_type_tenant;
ALTER TABLE managed_identities DROP COLUMN IF EXISTS tenant_id;
