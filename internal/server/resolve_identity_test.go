package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type resolveIdentityFallbackStore struct {
	resolveResult   store.ManagedIdentity
	resolveErr      error
	resolveByResult store.ManagedIdentity
	resolveByErr    error
	resolveByCalled bool
	resolveByInput  uuid.UUID
}

func (s *resolveIdentityFallbackStore) InsertManagedIdentity(_ context.Context, _ store.ManagedIdentity) error {
	return errors.New("unexpected insert managed identity")
}

func (s *resolveIdentityFallbackStore) DeleteManagedIdentity(_ context.Context, _ string) error {
	return errors.New("unexpected delete managed identity")
}

func (s *resolveIdentityFallbackStore) DeleteManagedIdentityByIdentityID(_ context.Context, _ uuid.UUID) error {
	return errors.New("unexpected delete managed identity by identity id")
}

func (s *resolveIdentityFallbackStore) ResolveIdentity(_ context.Context, _ string) (store.ManagedIdentity, error) {
	if s.resolveErr != nil {
		return store.ManagedIdentity{}, s.resolveErr
	}
	return s.resolveResult, nil
}

func (s *resolveIdentityFallbackStore) ResolveIdentityByIdentityID(_ context.Context, identityID uuid.UUID) (store.ManagedIdentity, error) {
	s.resolveByCalled = true
	s.resolveByInput = identityID
	if s.resolveByErr != nil {
		return store.ManagedIdentity{}, s.resolveByErr
	}
	return s.resolveByResult, nil
}

func (s *resolveIdentityFallbackStore) ListManagedIdentities(_ context.Context, _ store.ListFilter, _ int32, _ *store.PageCursor) (store.ListResult, error) {
	return store.ListResult{}, errors.New("unexpected list managed identities")
}

func (s *resolveIdentityFallbackStore) InsertServiceIdentity(_ context.Context, _ string, _ store.ServiceType, _ time.Time) error {
	return errors.New("unexpected insert service identity")
}

func (s *resolveIdentityFallbackStore) ExtendServiceIdentityLease(_ context.Context, _ string, _ time.Time) error {
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
	server := New(storeClient, &fakeZitiClient{}, time.Minute, false)

	resp, err := server.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{ZitiIdentityId: "ziti-identity"})
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if resp.GetWorkloadId() != workloadID.String() {
		t.Fatalf("expected workload id %s, got %s", workloadID, resp.GetWorkloadId())
	}
}

func TestResolveIdentityNameFallbackDisabled(t *testing.T) {
	ctx := context.Background()
	storeClient := &resolveIdentityFallbackStore{
		resolveErr:   store.ErrManagedIdentityNotFound,
		resolveByErr: errors.New("unexpected resolve identity by identity id"),
	}
	server := New(storeClient, &fakeZitiClient{}, time.Minute, false)

	_, err := server.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{ZitiIdentityId: fmt.Sprintf("agent-%s-suffix", uuid.New())})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	if storeClient.resolveByCalled {
		t.Fatalf("expected no resolve by identity id call")
	}
}

func TestResolveIdentityNameFallbackEnabled(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	storeClient := &resolveIdentityFallbackStore{
		resolveErr: store.ErrManagedIdentityNotFound,
		resolveByResult: store.ManagedIdentity{
			ZitiIdentityID: "ziti-identity",
			IdentityID:     agentID,
			IdentityType:   store.IdentityTypeAgent,
		},
	}
	server := New(storeClient, &fakeZitiClient{}, time.Minute, true)

	resp, err := server.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{ZitiIdentityId: fmt.Sprintf("agent-%s-suffix", agentID)})
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if resp.GetIdentityId() != agentID.String() {
		t.Fatalf("expected identity id %s, got %s", agentID, resp.GetIdentityId())
	}
	if !storeClient.resolveByCalled {
		t.Fatalf("expected resolve by identity id call")
	}
	if storeClient.resolveByInput != agentID {
		t.Fatalf("expected resolve by identity id %s, got %s", agentID, storeClient.resolveByInput)
	}
}

func TestResolveIdentityNameFallbackSkipsNonMatching(t *testing.T) {
	ctx := context.Background()
	storeClient := &resolveIdentityFallbackStore{
		resolveErr:   store.ErrManagedIdentityNotFound,
		resolveByErr: errors.New("unexpected resolve identity by identity id"),
	}
	server := New(storeClient, &fakeZitiClient{}, time.Minute, true)

	_, err := server.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{ZitiIdentityId: "agent-not-a-uuid"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	if storeClient.resolveByCalled {
		t.Fatalf("expected no resolve by identity id call")
	}
}
