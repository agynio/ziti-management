package store

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestListExpiredServiceIdentitiesGracePeriod(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pool mock: %v", err)
	}
	defer pool.Close()

	gracePeriod := 10 * time.Minute
	leaseExpiresAt := time.Now().Add(-gracePeriod - time.Minute)
	createdAt := time.Now().Add(-2 * time.Hour)

	rows := pgxmock.NewRows([]string{"ziti_identity_id", "service_type", "lease_expires_at", "created_at"}).
		AddRow("identity-1", ServiceTypeRunner, leaseExpiresAt, createdAt)

	pool.ExpectQuery(`SELECT ziti_identity_id, service_type, lease_expires_at, created_at FROM service_identities WHERE lease_expires_at < NOW\(\) - \$1::interval ORDER BY lease_expires_at ASC`).
		WithArgs(gracePeriod).
		WillReturnRows(rows)

	storeClient := NewStore(pool)
	identities, err := storeClient.ListExpiredServiceIdentities(ctx, gracePeriod)
	if err != nil {
		t.Fatalf("list expired service identities: %v", err)
	}

	if len(identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(identities))
	}
	identity := identities[0]
	if identity.ZitiIdentityID != "identity-1" {
		t.Fatalf("expected ziti identity id identity-1, got %s", identity.ZitiIdentityID)
	}
	if identity.ServiceType != ServiceTypeRunner {
		t.Fatalf("expected service type %v, got %v", ServiceTypeRunner, identity.ServiceType)
	}
	if !identity.LeaseExpiresAt.Equal(leaseExpiresAt) {
		t.Fatalf("expected lease expiry %v, got %v", leaseExpiresAt, identity.LeaseExpiresAt)
	}
	if !identity.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created at %v, got %v", createdAt, identity.CreatedAt)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
