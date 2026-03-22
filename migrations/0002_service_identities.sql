CREATE TABLE service_identities (
    ziti_identity_id TEXT PRIMARY KEY,
    service_type SMALLINT NOT NULL,
    lease_expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_service_identities_lease ON service_identities (lease_expires_at);
