package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
	"github.com/google/uuid"
)

type fakeManagedIdentityStore struct {
	managed               map[uuid.UUID]store.ManagedIdentity
	deleteCalls           []uuid.UUID
	deleteByIdentityIDErr error
}

func newFakeManagedIdentityStore() *fakeManagedIdentityStore {
	return &fakeManagedIdentityStore{managed: make(map[uuid.UUID]store.ManagedIdentity)}
}

func (f *fakeManagedIdentityStore) InsertManagedIdentity(_ context.Context, identity store.ManagedIdentity) error {
	if _, exists := f.managed[identity.IdentityID]; exists {
		return errors.New("managed identity already exists")
	}
	f.managed[identity.IdentityID] = identity
	return nil
}

func (f *fakeManagedIdentityStore) DeleteManagedIdentity(_ context.Context, _ string) error {
	return errors.New("unexpected delete managed identity")
}

func (f *fakeManagedIdentityStore) DeleteManagedIdentityByIdentityID(_ context.Context, identityID uuid.UUID) error {
	f.deleteCalls = append(f.deleteCalls, identityID)
	if f.deleteByIdentityIDErr != nil {
		return f.deleteByIdentityIDErr
	}
	delete(f.managed, identityID)
	return nil
}

func (f *fakeManagedIdentityStore) ResolveIdentity(_ context.Context, _ string) (store.ManagedIdentity, error) {
	return store.ManagedIdentity{}, errors.New("unexpected resolve identity")
}

func (f *fakeManagedIdentityStore) ResolveIdentityByIdentityID(_ context.Context, _ uuid.UUID) (store.ManagedIdentity, error) {
	return store.ManagedIdentity{}, errors.New("unexpected resolve identity by identity id")
}

func (f *fakeManagedIdentityStore) ListManagedIdentities(_ context.Context, _ store.ListFilter, _ int32, _ *store.PageCursor) (store.ListResult, error) {
	return store.ListResult{}, errors.New("unexpected list managed identities")
}

func (f *fakeManagedIdentityStore) InsertServiceIdentity(_ context.Context, _ string, _ store.ServiceType, _ time.Time) error {
	return errors.New("unexpected insert service identity")
}

func (f *fakeManagedIdentityStore) ExtendServiceIdentityLease(_ context.Context, _ string, _ time.Time) error {
	return errors.New("unexpected extend service identity")
}

type fakeZitiClient struct {
	appCount          int
	runnerCount       int
	deleteIdentityIDs []string
}

func (f *fakeZitiClient) CreateAgentIdentity(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", errors.New("unexpected create agent identity")
}

func (f *fakeZitiClient) CreateAndEnrollAppIdentity(_ context.Context, _ uuid.UUID, _ string) (string, []byte, error) {
	f.appCount++
	zitiID := fmt.Sprintf("app-ziti-%d", f.appCount)
	return zitiID, []byte(fmt.Sprintf("app-json-%d", f.appCount)), nil
}

func (f *fakeZitiClient) CreateAndEnrollRunnerIdentity(_ context.Context, _ uuid.UUID, _ []string) (string, []byte, error) {
	f.runnerCount++
	zitiID := fmt.Sprintf("runner-ziti-%d", f.runnerCount)
	return zitiID, []byte(fmt.Sprintf("runner-json-%d", f.runnerCount)), nil
}

func (f *fakeZitiClient) CreateAndEnrollServiceIdentity(_ context.Context, _ string, _ []string) (string, []byte, error) {
	return "", nil, errors.New("unexpected create service identity")
}

func (f *fakeZitiClient) CreateService(_ context.Context, _ string, _ []string) (string, error) {
	return "", errors.New("unexpected create service")
}

func (f *fakeZitiClient) CreateServiceWithConfigs(_ context.Context, _ string, _ []string, _ *ziti.HostV1ConfigData, _ *ziti.InterceptV1ConfigData) (string, error) {
	return "", errors.New("unexpected create service with configs")
}

func (f *fakeZitiClient) CreateServicePolicy(_ context.Context, _ string, _ string, _ []string, _ []string) (string, error) {
	return "", errors.New("unexpected create service policy")
}

func (f *fakeZitiClient) CreateDeviceIdentity(_ context.Context, _ uuid.UUID, _ string) (string, string, error) {
	return "", "", errors.New("unexpected create device identity")
}

func (f *fakeZitiClient) DeleteIdentity(_ context.Context, zitiID string) error {
	f.deleteIdentityIDs = append(f.deleteIdentityIDs, zitiID)
	return nil
}

func (f *fakeZitiClient) DeleteService(_ context.Context, _ string) error {
	return nil
}

func (f *fakeZitiClient) DeleteServicePolicy(_ context.Context, _ string) error {
	return nil
}

func TestCreateAppIdentityAllowsReenroll(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute)

	request := &zitimanagementv1.CreateAppIdentityRequest{
		IdentityId: appID.String(),
		Slug:       "app-slug",
	}

	firstResp, err := server.CreateAppIdentity(ctx, request)
	if err != nil {
		t.Fatalf("create app identity: %v", err)
	}
	secondResp, err := server.CreateAppIdentity(ctx, request)
	if err != nil {
		t.Fatalf("create app identity again: %v", err)
	}

	if len(storeClient.deleteCalls) != 2 {
		t.Fatalf("expected 2 delete calls, got %d", len(storeClient.deleteCalls))
	}
	for _, deletedID := range storeClient.deleteCalls {
		if deletedID != appID {
			t.Fatalf("expected delete call for %s, got %s", appID, deletedID)
		}
	}

	stored, ok := storeClient.managed[appID]
	if !ok {
		t.Fatalf("expected managed identity for %s", appID)
	}
	if stored.ZitiIdentityID != secondResp.GetZitiIdentityId() {
		t.Fatalf("expected stored ziti id %q, got %q", secondResp.GetZitiIdentityId(), stored.ZitiIdentityID)
	}
	if firstResp.GetZitiIdentityId() == secondResp.GetZitiIdentityId() {
		t.Fatalf("expected distinct ziti ids for reenroll")
	}
}

func TestCreateAppIdentityDeleteFailureCleansUp(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	storeClient := newFakeManagedIdentityStore()
	storeClient.deleteByIdentityIDErr = errors.New("delete failed")
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute)

	request := &zitimanagementv1.CreateAppIdentityRequest{
		IdentityId: appID.String(),
		Slug:       "app-slug",
	}

	_, err := server.CreateAppIdentity(ctx, request)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "delete managed identity") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zitiClient.deleteIdentityIDs) != 1 {
		t.Fatalf("expected 1 cleanup call, got %d", len(zitiClient.deleteIdentityIDs))
	}
	if zitiClient.deleteIdentityIDs[0] != "app-ziti-1" {
		t.Fatalf("expected cleanup for app-ziti-1, got %s", zitiClient.deleteIdentityIDs[0])
	}
	if len(storeClient.managed) != 0 {
		t.Fatalf("expected no managed identities persisted")
	}
}

func TestCreateRunnerIdentityAllowsReenroll(t *testing.T) {
	ctx := context.Background()
	runnerID := uuid.New()
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute)

	request := &zitimanagementv1.CreateRunnerIdentityRequest{
		RunnerId:       runnerID.String(),
		RoleAttributes: []string{"runner"},
	}

	firstResp, err := server.CreateRunnerIdentity(ctx, request)
	if err != nil {
		t.Fatalf("create runner identity: %v", err)
	}
	secondResp, err := server.CreateRunnerIdentity(ctx, request)
	if err != nil {
		t.Fatalf("create runner identity again: %v", err)
	}

	if len(storeClient.deleteCalls) != 2 {
		t.Fatalf("expected 2 delete calls, got %d", len(storeClient.deleteCalls))
	}
	for _, deletedID := range storeClient.deleteCalls {
		if deletedID != runnerID {
			t.Fatalf("expected delete call for %s, got %s", runnerID, deletedID)
		}
	}

	stored, ok := storeClient.managed[runnerID]
	if !ok {
		t.Fatalf("expected managed identity for %s", runnerID)
	}
	if stored.ZitiIdentityID != secondResp.GetZitiIdentityId() {
		t.Fatalf("expected stored ziti id %q, got %q", secondResp.GetZitiIdentityId(), stored.ZitiIdentityID)
	}
	if firstResp.GetZitiIdentityId() == secondResp.GetZitiIdentityId() {
		t.Fatalf("expected distinct ziti ids for reenroll")
	}
}

func TestCreateRunnerIdentityDeleteFailureCleansUp(t *testing.T) {
	ctx := context.Background()
	runnerID := uuid.New()
	storeClient := newFakeManagedIdentityStore()
	storeClient.deleteByIdentityIDErr = errors.New("delete failed")
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute)

	request := &zitimanagementv1.CreateRunnerIdentityRequest{
		RunnerId:       runnerID.String(),
		RoleAttributes: []string{"runner"},
	}

	_, err := server.CreateRunnerIdentity(ctx, request)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "delete managed identity") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zitiClient.deleteIdentityIDs) != 1 {
		t.Fatalf("expected 1 cleanup call, got %d", len(zitiClient.deleteIdentityIDs))
	}
	if zitiClient.deleteIdentityIDs[0] != "runner-ziti-1" {
		t.Fatalf("expected cleanup for runner-ziti-1, got %s", zitiClient.deleteIdentityIDs[0])
	}
	if len(storeClient.managed) != 0 {
		t.Fatalf("expected no managed identities persisted")
	}
}
