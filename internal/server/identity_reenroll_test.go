package server

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	deleteServiceIDs  []string
	agentCalls        []agentCall
	createAgentErr    error
	patchIdentityErr  error
	patchIdentityCall *patchIdentityCall
	tunnelExpiresAt   time.Time
}

type patchIdentityCall struct {
	zitiID string
	add    []string
	remove []string
}

type agentCall struct {
	agentID    uuid.UUID
	workloadID uuid.UUID
}

func (f *fakeZitiClient) CreateAgentIdentity(_ context.Context, agentID, workloadID uuid.UUID) (string, string, error) {
	call := agentCall{agentID: agentID, workloadID: workloadID}
	f.agentCalls = append(f.agentCalls, call)
	if f.createAgentErr != nil {
		return "", "", f.createAgentErr
	}
	return fmt.Sprintf("agent-ziti-%d", len(f.agentCalls)), fmt.Sprintf("agent-jwt-%d", len(f.agentCalls)), nil
}

func (f *fakeZitiClient) CreateAgentIdentityWithOptions(ctx context.Context, agentID, workloadID uuid.UUID, _ []string, _ map[string]string) (string, string, error) {
	return f.CreateAgentIdentity(ctx, agentID, workloadID)
}

func (f *fakeZitiClient) CreateAndEnrollAppIdentity(_ context.Context, _ uuid.UUID, _ string) (string, []byte, error) {
	f.appCount++
	zitiID := fmt.Sprintf("app-ziti-%d", f.appCount)
	return zitiID, []byte(fmt.Sprintf("app-json-%d", f.appCount)), nil
}

func (f *fakeZitiClient) CreateAndEnrollAppIdentityWithOptions(ctx context.Context, appID uuid.UUID, slug string, _ []string, _ map[string]string) (string, []byte, error) {
	return f.CreateAndEnrollAppIdentity(ctx, appID, slug)
}

func (f *fakeZitiClient) CreateAndEnrollRunnerIdentity(_ context.Context, _ uuid.UUID, _ []string) (string, []byte, error) {
	f.runnerCount++
	zitiID := fmt.Sprintf("runner-ziti-%d", f.runnerCount)
	return zitiID, []byte(fmt.Sprintf("runner-json-%d", f.runnerCount)), nil
}

func (f *fakeZitiClient) CreateAndEnrollRunnerIdentityWithTags(ctx context.Context, runnerID uuid.UUID, roleAttributes []string, _ map[string]string) (string, []byte, error) {
	return f.CreateAndEnrollRunnerIdentity(ctx, runnerID, roleAttributes)
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

func (f *fakeZitiClient) CreateServiceWithConfigsAndTags(ctx context.Context, name string, roleAttributes []string, hostV1 *ziti.HostV1ConfigData, interceptV1 *ziti.InterceptV1ConfigData, _ map[string]string) (string, error) {
	return f.CreateServiceWithConfigs(ctx, name, roleAttributes, hostV1, interceptV1)
}

func (f *fakeZitiClient) CreateServicePolicy(_ context.Context, _ string, _ string, _ []string, _ []string) (string, error) {
	return "", errors.New("unexpected create service policy")
}

func (f *fakeZitiClient) CreateServicePolicyWithTags(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string, _ map[string]string) (string, error) {
	return f.CreateServicePolicy(ctx, name, policyType, identityRoles, serviceRoles)
}

func (f *fakeZitiClient) CreateDeviceIdentity(_ context.Context, _ uuid.UUID, _ string) (string, string, error) {
	return "", "", errors.New("unexpected create device identity")
}

func (f *fakeZitiClient) CreateDeviceIdentityWithOptions(ctx context.Context, userIdentityID uuid.UUID, name string, _ []string, _ map[string]string) (string, string, error) {
	return f.CreateDeviceIdentity(ctx, userIdentityID, name)
}

func (f *fakeZitiClient) CreateTunnelIdentity(_ context.Context, _, _ string, _ map[string]string) (string, ziti.EnrollmentJWT, error) {
	if f.tunnelExpiresAt.IsZero() {
		return "", ziti.EnrollmentJWT{}, errors.New("unexpected create tunnel identity")
	}
	return "tunnel-ziti", ziti.EnrollmentJWT{Token: "tunnel-jwt", ExpiresAt: f.tunnelExpiresAt}, nil
}

func (f *fakeZitiClient) DeleteIdentity(_ context.Context, zitiID string) error {
	f.deleteIdentityIDs = append(f.deleteIdentityIDs, zitiID)
	return nil
}

func (f *fakeZitiClient) DeleteService(_ context.Context, serviceID string) error {
	f.deleteServiceIDs = append(f.deleteServiceIDs, serviceID)
	return nil
}

func (f *fakeZitiClient) DeleteServicePolicy(_ context.Context, _ string) error {
	return nil
}

func (f *fakeZitiClient) PatchIdentityRoleAttributes(_ context.Context, zitiID string, add, remove []string) error {
	f.patchIdentityCall = &patchIdentityCall{
		zitiID: zitiID,
		add:    append([]string(nil), add...),
		remove: append([]string(nil), remove...),
	}
	return f.patchIdentityErr
}

func (f *fakeZitiClient) GetIdentityLiveness(_ context.Context, _ string) (ziti.IdentityLiveness, error) {
	return ziti.IdentityLiveness{}, errors.New("unexpected get identity liveness")
}

func (f *fakeZitiClient) ListServicesByTag(_ context.Context, _ map[string]string, _ int32, _ string) (ziti.ListResult[ziti.OpenZitiService], error) {
	return ziti.ListResult[ziti.OpenZitiService]{}, errors.New("unexpected list services by tag")
}

func (f *fakeZitiClient) ListIdentitiesByTag(_ context.Context, _ map[string]string, _ int32, _ string) (ziti.ListResult[ziti.OpenZitiIdentity], error) {
	return ziti.ListResult[ziti.OpenZitiIdentity]{}, errors.New("unexpected list identities by tag")
}

func (f *fakeZitiClient) ListServicePoliciesByTag(_ context.Context, _ map[string]string, _ int32, _ string) (ziti.ListResult[ziti.OpenZitiServicePolicy], error) {
	return ziti.ListResult[ziti.OpenZitiServicePolicy]{}, errors.New("unexpected list service policies by tag")
}

func (f *fakeZitiClient) UpdateService(_ context.Context, _ string, _ *ziti.HostV1ConfigData, _ *ziti.InterceptV1ConfigData, _ map[string]string, _ bool) (ziti.OpenZitiService, error) {
	return ziti.OpenZitiService{}, errors.New("unexpected update service")
}

func TestCreateAppIdentityAllowsReenroll(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

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
	server := New(storeClient, zitiClient, time.Minute, false)

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
	server := New(storeClient, zitiClient, time.Minute, false)

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
	server := New(storeClient, zitiClient, time.Minute, false)

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

func TestCreateAgentIdentityStoresWorkloadID(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	workloadID := uuid.New()
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	request := &zitimanagementv1.CreateAgentIdentityRequest{
		AgentId:    agentID.String(),
		WorkloadId: workloadID.String(),
	}

	resp, err := server.CreateAgentIdentity(ctx, request)
	if err != nil {
		t.Fatalf("create agent identity: %v", err)
	}
	if resp.GetZitiIdentityId() != "agent-ziti-1" {
		t.Fatalf("expected agent-ziti-1, got %q", resp.GetZitiIdentityId())
	}
	if len(zitiClient.agentCalls) != 1 {
		t.Fatalf("expected 1 agent call, got %d", len(zitiClient.agentCalls))
	}
	if zitiClient.agentCalls[0].agentID != agentID {
		t.Fatalf("expected agent id %s, got %s", agentID, zitiClient.agentCalls[0].agentID)
	}
	if zitiClient.agentCalls[0].workloadID != workloadID {
		t.Fatalf("expected workload id %s, got %s", workloadID, zitiClient.agentCalls[0].workloadID)
	}
	stored, ok := storeClient.managed[agentID]
	if !ok {
		t.Fatalf("expected managed identity for %s", agentID)
	}
	if stored.WorkloadID == nil {
		t.Fatalf("expected workload id to be stored")
	}
	if *stored.WorkloadID != workloadID {
		t.Fatalf("expected workload id %s, got %s", workloadID, *stored.WorkloadID)
	}
}

func TestCreateTunnelIdentityReturnsJWTExpiry(t *testing.T) {
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{tunnelExpiresAt: expiresAt}
	server := New(storeClient, zitiClient, time.Minute, false)

	resp, err := server.CreateTunnelIdentity(ctx, &zitimanagementv1.CreateTunnelIdentityRequest{
		NetworkId:          "network-1",
		TunnelCredentialId: "credential-1",
	})
	if err != nil {
		t.Fatalf("create tunnel identity: %v", err)
	}
	if resp.GetZitiIdentityId() != "tunnel-ziti" || resp.GetEnrollmentJwt() != "tunnel-jwt" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.GetEnrollmentJwtExpiresAt() == nil || !resp.GetEnrollmentJwtExpiresAt().AsTime().Equal(expiresAt) {
		t.Fatalf("expected expiry %s, got %#v", expiresAt, resp.GetEnrollmentJwtExpiresAt())
	}
}

func TestPatchIdentityRoleAttributesDelegatesToZiti(t *testing.T) {
	ctx := context.Background()
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	_, err := server.PatchIdentityRoleAttributes(ctx, &zitimanagementv1.PatchIdentityRoleAttributesRequest{
		ZitiIdentityId: "identity-id",
		Add:            []string{"group-one"},
		Remove:         []string{"group-two"},
	})
	if err != nil {
		t.Fatalf("patch identity role attributes: %v", err)
	}
	if zitiClient.patchIdentityCall == nil || zitiClient.patchIdentityCall.zitiID != "identity-id" || !reflect.DeepEqual(zitiClient.patchIdentityCall.add, []string{"group-one"}) || !reflect.DeepEqual(zitiClient.patchIdentityCall.remove, []string{"group-two"}) {
		t.Fatalf("unexpected patch call: %#v", zitiClient.patchIdentityCall)
	}
}

func TestPatchIdentityRoleAttributesMapsNotFound(t *testing.T) {
	ctx := context.Background()
	storeClient := newFakeManagedIdentityStore()
	zitiClient := &fakeZitiClient{patchIdentityErr: ziti.ErrIdentityNotFound}
	server := New(storeClient, zitiClient, time.Minute, false)

	_, err := server.PatchIdentityRoleAttributes(ctx, &zitimanagementv1.PatchIdentityRoleAttributesRequest{
		ZitiIdentityId: "identity-id",
		Add:            []string{"group-one"},
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
