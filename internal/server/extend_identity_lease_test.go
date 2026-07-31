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

type extendLeaseStore struct {
	extendErr error
	extended  bool
}

func (s *extendLeaseStore) InsertManagedIdentity(context.Context, store.ManagedIdentity) error {
	return errors.New("unexpected insert managed identity")
}

func (s *extendLeaseStore) DeleteManagedIdentity(context.Context, string) error {
	return errors.New("unexpected delete managed identity")
}

func (s *extendLeaseStore) DeleteManagedIdentityByIdentityID(context.Context, uuid.UUID) error {
	return errors.New("unexpected delete managed identity by identity id")
}

func (s *extendLeaseStore) ResolveIdentity(context.Context, string) (store.ManagedIdentity, error) {
	return store.ManagedIdentity{}, errors.New("unexpected resolve identity")
}

func (s *extendLeaseStore) ResolveIdentityByIdentityID(context.Context, uuid.UUID) (store.ManagedIdentity, error) {
	return store.ManagedIdentity{}, errors.New("unexpected resolve identity by identity id")
}

func (s *extendLeaseStore) ListManagedIdentities(context.Context, store.ListFilter, int32, *store.PageCursor) (store.ListResult, error) {
	return store.ListResult{}, errors.New("unexpected list managed identities")
}

func (s *extendLeaseStore) InsertServiceIdentity(context.Context, string, store.ServiceType, time.Time) error {
	return errors.New("unexpected insert service identity")
}

func (s *extendLeaseStore) ExtendServiceIdentityLease(context.Context, string, time.Time) error {
	s.extended = true
	return s.extendErr
}

// TestExtendIdentityLeaseMapsNotFound asserts the contract the service fail-fast
// logic depends on: a lease extension for an identity with no record returns
// codes.NotFound so the caller can treat it as definitive identity loss.
func TestExtendIdentityLeaseMapsNotFound(t *testing.T) {
	storeClient := &extendLeaseStore{extendErr: store.ErrServiceIdentityNotFound}
	server := New(storeClient, &fakeZitiClient{}, time.Minute, false)

	_, err := server.ExtendIdentityLease(context.Background(), &zitimanagementv1.ExtendIdentityLeaseRequest{
		ZitiIdentityId: "gone",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
	if !storeClient.extended {
		t.Fatal("expected store extension to be attempted")
	}
}

func TestExtendIdentityLeaseSucceeds(t *testing.T) {
	storeClient := &extendLeaseStore{}
	server := New(storeClient, &fakeZitiClient{}, time.Minute, false)

	if _, err := server.ExtendIdentityLease(context.Background(), &zitimanagementv1.ExtendIdentityLeaseRequest{
		ZitiIdentityId: "present",
	}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !storeClient.extended {
		t.Fatal("expected store extension to be attempted")
	}
}

func TestExtendIdentityLeaseRequiresIdentityID(t *testing.T) {
	server := New(&extendLeaseStore{}, &fakeZitiClient{}, time.Minute, false)

	if _, err := server.ExtendIdentityLease(context.Background(), &zitimanagementv1.ExtendIdentityLeaseRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}
