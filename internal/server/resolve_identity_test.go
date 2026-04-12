package server

import (
	"context"
	"errors"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/google/uuid"
)

type resolveIdentityStore struct {
	identity store.ManagedIdentity
}

func (s *resolveIdentityStore) InsertManagedIdentity(_ context.Context, _ store.ManagedIdentity) error {
	return errors.New("unexpected insert managed identity")
}

func (s *resolveIdentityStore) DeleteManagedIdentity(_ context.Context, _ string) error {
	return errors.New("unexpected delete managed identity")
}

func (s *resolveIdentityStore) DeleteManagedIdentityByIdentityID(_ context.Context, _ uuid.UUID) error {
	return errors.New("unexpected delete managed identity by identity id")
}

func (s *resolveIdentityStore) ResolveIdentity(_ context.Context, zitiIdentityID string) (store.ManagedIdentity, error) {
	if zitiIdentityID != s.identity.ZitiIdentityID {
		return store.ManagedIdentity{}, store.ErrManagedIdentityNotFound
	}
	return s.identity, nil
}

func (s *resolveIdentityStore) ResolveIdentityByIdentityID(_ context.Context, _ uuid.UUID) (store.ManagedIdentity, error) {
	return store.ManagedIdentity{}, errors.New("unexpected resolve identity by identity id")
}

func (s *resolveIdentityStore) ListManagedIdentities(_ context.Context, _ store.ListFilter, _ int32, _ *store.PageCursor) (store.ListResult, error) {
	return store.ListResult{}, errors.New("unexpected list managed identities")
}

func (s *resolveIdentityStore) InsertServiceIdentity(_ context.Context, _ string, _ store.ServiceType, _ time.Time) error {
	return errors.New("unexpected insert service identity")
}

func (s *resolveIdentityStore) ExtendServiceIdentityLease(_ context.Context, _ string, _ time.Time) error {
	return errors.New("unexpected extend service identity")
}

func TestResolveIdentityIncludesWorkloadID(t *testing.T) {
	ctx := context.Background()
	identityID := uuid.New()
	workloadID := uuid.New()
	storeClient := &resolveIdentityStore{
		identity: store.ManagedIdentity{
			ZitiIdentityID: "ziti-identity",
			IdentityID:     identityID,
			WorkloadID:     &workloadID,
			IdentityType:   store.IdentityTypeAgent,
		},
	}
	server := New(storeClient, &fakeZitiClient{}, time.Minute)

	resp, err := server.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{ZitiIdentityId: "ziti-identity"})
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if resp.GetWorkloadId() != workloadID.String() {
		t.Fatalf("expected workload id %s, got %s", workloadID, resp.GetWorkloadId())
	}
}
