package server

import (
	"context"
	"errors"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type deleteAppIdentityStore struct {
	identity    store.ManagedIdentity
	resolveErr  error
	deleteErr   error
	deleteCalls []string
}

func (s *deleteAppIdentityStore) InsertManagedIdentity(_ context.Context, _ store.ManagedIdentity) error {
	return errors.New("unexpected insert managed identity")
}

func (s *deleteAppIdentityStore) DeleteManagedIdentity(_ context.Context, zitiIdentityID string) error {
	s.deleteCalls = append(s.deleteCalls, zitiIdentityID)
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return nil
}

func (s *deleteAppIdentityStore) DeleteManagedIdentityByIdentityID(_ context.Context, _ uuid.UUID) error {
	return errors.New("unexpected delete managed identity by identity id")
}

func (s *deleteAppIdentityStore) ResolveIdentity(_ context.Context, _ string) (store.ManagedIdentity, error) {
	return store.ManagedIdentity{}, errors.New("unexpected resolve identity")
}

func (s *deleteAppIdentityStore) ResolveIdentityByIdentityID(_ context.Context, _ uuid.UUID) (store.ManagedIdentity, error) {
	if s.resolveErr != nil {
		return store.ManagedIdentity{}, s.resolveErr
	}
	return s.identity, nil
}

func (s *deleteAppIdentityStore) ListManagedIdentities(_ context.Context, _ store.ListFilter, _ int32, _ *store.PageCursor) (store.ListResult, error) {
	return store.ListResult{}, errors.New("unexpected list managed identities")
}

func (s *deleteAppIdentityStore) InsertServiceIdentity(_ context.Context, _ string, _ store.ServiceType, _ time.Time) error {
	return errors.New("unexpected insert service identity")
}

func (s *deleteAppIdentityStore) ExtendServiceIdentityLease(_ context.Context, _ string, _ time.Time) error {
	return errors.New("unexpected extend service identity")
}

func TestDeleteAppIdentityDeletesServiceWhenIdentityMissing(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	storeClient := &deleteAppIdentityStore{resolveErr: store.ErrManagedIdentityNotFound}
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	serviceID := "service-123"
	_, err := server.DeleteAppIdentity(ctx, &zitimanagementv1.DeleteAppIdentityRequest{
		IdentityId:    appID.String(),
		ZitiServiceId: serviceID,
	})
	if err != nil {
		t.Fatalf("delete app identity: %v", err)
	}
	if len(zitiClient.deleteServiceIDs) != 1 {
		t.Fatalf("expected 1 service delete call, got %d", len(zitiClient.deleteServiceIDs))
	}
	if zitiClient.deleteServiceIDs[0] != serviceID {
		t.Fatalf("expected service delete %q, got %q", serviceID, zitiClient.deleteServiceIDs[0])
	}
	if len(zitiClient.deleteIdentityIDs) != 0 {
		t.Fatalf("expected no identity deletes, got %d", len(zitiClient.deleteIdentityIDs))
	}
	if len(storeClient.deleteCalls) != 0 {
		t.Fatalf("expected no managed identity deletes, got %d", len(storeClient.deleteCalls))
	}
}

func TestDeleteAppIdentityUsesStoredServiceID(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	serviceID := "service-456"
	zitiID := "ziti-identity"
	storeClient := &deleteAppIdentityStore{
		identity: store.ManagedIdentity{
			ZitiIdentityID: zitiID,
			IdentityID:     appID,
			ZitiServiceID:  &serviceID,
		},
	}
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	_, err := server.DeleteAppIdentity(ctx, &zitimanagementv1.DeleteAppIdentityRequest{
		IdentityId: appID.String(),
	})
	if err != nil {
		t.Fatalf("delete app identity: %v", err)
	}
	if len(storeClient.deleteCalls) != 1 {
		t.Fatalf("expected 1 managed identity delete, got %d", len(storeClient.deleteCalls))
	}
	if storeClient.deleteCalls[0] != zitiID {
		t.Fatalf("expected managed identity delete %q, got %q", zitiID, storeClient.deleteCalls[0])
	}
	if len(zitiClient.deleteIdentityIDs) != 1 {
		t.Fatalf("expected 1 identity delete, got %d", len(zitiClient.deleteIdentityIDs))
	}
	if zitiClient.deleteIdentityIDs[0] != zitiID {
		t.Fatalf("expected identity delete %q, got %q", zitiID, zitiClient.deleteIdentityIDs[0])
	}
	if len(zitiClient.deleteServiceIDs) != 1 {
		t.Fatalf("expected 1 service delete call, got %d", len(zitiClient.deleteServiceIDs))
	}
	if zitiClient.deleteServiceIDs[0] != serviceID {
		t.Fatalf("expected service delete %q, got %q", serviceID, zitiClient.deleteServiceIDs[0])
	}
}

func TestDeleteAppIdentityRequiresServiceIDWithoutMapping(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	storeClient := &deleteAppIdentityStore{resolveErr: store.ErrManagedIdentityNotFound}
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	_, err := server.DeleteAppIdentity(ctx, &zitimanagementv1.DeleteAppIdentityRequest{
		IdentityId: appID.String(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument error, got %v", err)
	}
	if len(zitiClient.deleteServiceIDs) != 0 {
		t.Fatalf("expected no service delete calls, got %d", len(zitiClient.deleteServiceIDs))
	}
}
