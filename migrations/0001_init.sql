CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE managed_identities (
    ziti_identity_id TEXT PRIMARY KEY,
    identity_id UUID NOT NULL,
    identity_type SMALLINT NOT NULL,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_managed_identities_type ON managed_identities (identity_type);
CREATE INDEX idx_managed_identities_tenant ON managed_identities (tenant_id);
CREATE INDEX idx_managed_identities_type_tenant ON managed_identities (identity_type, tenant_id);
