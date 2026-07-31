package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	identityv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/identity/v1"
	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/agynio/ziti-management/internal/id"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
)

type managedIdentityStore interface {
	InsertManagedIdentity(ctx context.Context, identity store.ManagedIdentity) error
	DeleteManagedIdentity(ctx context.Context, zitiIdentityID string) error
	DeleteManagedIdentityByIdentityID(ctx context.Context, identityID uuid.UUID) error
	ResolveIdentity(ctx context.Context, zitiIdentityID string) (store.ManagedIdentity, error)
	ResolveIdentityByIdentityID(ctx context.Context, identityID uuid.UUID) (store.ManagedIdentity, error)
	ListManagedIdentities(ctx context.Context, filter store.ListFilter, pageSize int32, cursor *store.PageCursor) (store.ListResult, error)
	InsertServiceIdentity(ctx context.Context, zitiIdentityID string, serviceType store.ServiceType, leaseExpiresAt time.Time) error
	ExtendServiceIdentityLease(ctx context.Context, zitiIdentityID string, leaseExpiresAt time.Time) error
}

type zitiClient interface {
	CreateAgentIdentity(ctx context.Context, agentID, workloadID uuid.UUID) (string, string, error)
	CreateAgentIdentityWithOptions(ctx context.Context, agentID, workloadID uuid.UUID, additionalRoleAttributes []string, tags map[string]string) (string, string, error)
	CreateSandboxIdentity(ctx context.Context, sandboxID, ownerID, environmentID uuid.UUID, organizationID string, workloadID uuid.UUID, additionalRoleAttributes []string, tags map[string]string) (string, string, error)
	CreateAndEnrollAppIdentity(ctx context.Context, appID uuid.UUID, slug string) (string, []byte, error)
	CreateAndEnrollAppIdentityWithOptions(ctx context.Context, appID uuid.UUID, slug string, additionalRoleAttributes []string, tags map[string]string) (string, []byte, error)
	CreateAndEnrollRunnerIdentity(ctx context.Context, runnerID uuid.UUID, roleAttributes []string) (string, []byte, error)
	CreateAndEnrollRunnerIdentityWithTags(ctx context.Context, runnerID uuid.UUID, roleAttributes []string, tags map[string]string) (string, []byte, error)
	CreateAndEnrollServiceIdentity(ctx context.Context, name string, roleAttributes []string) (string, []byte, error)
	CreateService(ctx context.Context, name string, roleAttributes []string) (string, error)
	CreateServiceWithConfigs(ctx context.Context, name string, roleAttributes []string, hostV1 *ziti.HostV1ConfigData, interceptV1 *ziti.InterceptV1ConfigData) (string, error)
	CreateServiceWithConfigsAndTags(ctx context.Context, name string, roleAttributes []string, hostV1 *ziti.HostV1ConfigData, interceptV1 *ziti.InterceptV1ConfigData, tags map[string]string) (string, error)
	GetService(ctx context.Context, serviceID string) (ziti.OpenZitiService, error)
	ListServices(ctx context.Context, filter ziti.ServiceListFilter, pageSize int32, pageToken string) (ziti.ListResult[ziti.OpenZitiService], error)
	CreateServicePolicy(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string) (string, error)
	CreateServicePolicyWithTags(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string, tags map[string]string) (string, error)
	GetServicePolicy(ctx context.Context, policyID string) (ziti.OpenZitiServicePolicy, error)
	ListServicePolicies(ctx context.Context, filter ziti.ServicePolicyListFilter, pageSize int32, pageToken string) (ziti.ListResult[ziti.OpenZitiServicePolicy], error)
	CreateDeviceIdentity(ctx context.Context, userIdentityID uuid.UUID, name string) (string, string, error)
	CreateDeviceIdentityWithOptions(ctx context.Context, userIdentityID uuid.UUID, name string, additionalRoleAttributes []string, tags map[string]string) (string, string, error)
	CreateTunnelIdentity(ctx context.Context, networkID, tunnelCredentialID string, tags map[string]string) (string, ziti.EnrollmentJWT, error)
	DeleteIdentity(ctx context.Context, zitiIdentityID string) error
	DeleteService(ctx context.Context, serviceID string) error
	DeleteServicePolicy(ctx context.Context, policyID string) error
	PatchIdentityRoleAttributes(ctx context.Context, zitiIdentityID string, add, remove []string) error
	GetIdentityLiveness(ctx context.Context, zitiIdentityID string) (ziti.IdentityLiveness, error)
	ListServicesByTag(ctx context.Context, tags map[string]string, pageSize int32, pageToken string) (ziti.ListResult[ziti.OpenZitiService], error)
	ListIdentitiesByTag(ctx context.Context, tags map[string]string, pageSize int32, pageToken string) (ziti.ListResult[ziti.OpenZitiIdentity], error)
	ListServicePoliciesByTag(ctx context.Context, tags map[string]string, pageSize int32, pageToken string) (ziti.ListResult[ziti.OpenZitiServicePolicy], error)
	UpdateService(ctx context.Context, serviceID string, hostV1 *ziti.HostV1ConfigData, interceptV1 *ziti.InterceptV1ConfigData, tags map[string]string, updateTags bool) (ziti.OpenZitiService, error)
}

type Server struct {
	zitimanagementv1.UnimplementedZitiManagementServiceServer
	store                   managedIdentityStore
	ziti                    zitiClient
	serviceIdentityLeaseTTL time.Duration
	resolveIdentityName     bool
}

func (s *Server) cleanupZitiIdentity(ctx context.Context, zitiID, label string) {
	if err := s.ziti.DeleteIdentity(ctx, zitiID); err != nil && !errors.Is(err, ziti.ErrIdentityNotFound) {
		log.Printf("failed to cleanup %s %s: %v", label, zitiID, err)
	}
}

func New(store managedIdentityStore, zitiClient zitiClient, serviceIdentityLeaseTTL time.Duration, resolveIdentityName bool) *Server {
	return &Server{
		store:                   store,
		ziti:                    zitiClient,
		serviceIdentityLeaseTTL: serviceIdentityLeaseTTL,
		resolveIdentityName:     resolveIdentityName,
	}
}

func (s *Server) CreateAgentIdentity(ctx context.Context, req *zitimanagementv1.CreateAgentIdentityRequest) (*zitimanagementv1.CreateAgentIdentityResponse, error) {
	agentID, err := parseUUID(req.GetAgentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "agent_id: %v", err)
	}

	workloadID, err := parseUUID(req.GetWorkloadId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "workload_id: %v", err)
	}

	zitiID, jwt, err := s.ziti.CreateAgentIdentityWithOptions(ctx, agentID, workloadID, req.GetAdditionalRoleAttributes(), req.GetTags())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create ziti identity: %v", err)
	}

	identity := store.ManagedIdentity{
		ZitiIdentityID: zitiID,
		IdentityID:     agentID,
		WorkloadID:     &workloadID,
		IdentityType:   store.IdentityTypeAgentInstance,
	}
	if err := s.store.InsertManagedIdentity(ctx, identity); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "ziti identity")
		return nil, status.Errorf(codes.Internal, "insert managed identity: %v", err)
	}

	return &zitimanagementv1.CreateAgentIdentityResponse{
		ZitiIdentityId: zitiID,
		EnrollmentJwt:  jwt,
	}, nil
}

func (s *Server) CreateSandboxIdentity(ctx context.Context, req *zitimanagementv1.CreateSandboxIdentityRequest) (*zitimanagementv1.CreateSandboxIdentityResponse, error) {
	sandboxID, err := parseUUID(req.GetSandboxId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "sandbox_id: %v", err)
	}
	ownerID, err := parseUUID(req.GetOwnerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "owner_id: %v", err)
	}
	environmentID, err := parseUUID(req.GetEnvironmentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "environment_id: %v", err)
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if organizationID == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}
	workloadID, err := parseUUID(req.GetWorkloadId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "workload_id: %v", err)
	}

	zitiID, jwt, err := s.ziti.CreateSandboxIdentity(ctx, sandboxID, ownerID, environmentID, organizationID, workloadID, req.GetAdditionalRoleAttributes(), req.GetTags())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create sandbox ziti identity: %v", err)
	}

	identity := store.ManagedIdentity{
		ZitiIdentityID: zitiID,
		IdentityID:     sandboxID,
		WorkloadID:     &workloadID,
		IdentityType:   store.IdentityTypeSandbox,
	}
	if err := s.store.InsertManagedIdentity(ctx, identity); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "sandbox identity")
		return nil, status.Errorf(codes.Internal, "insert managed identity: %v", err)
	}

	return &zitimanagementv1.CreateSandboxIdentityResponse{
		ZitiIdentityId: zitiID,
		EnrollmentJwt:  jwt,
	}, nil
}

func (s *Server) CreateAppIdentity(ctx context.Context, req *zitimanagementv1.CreateAppIdentityRequest) (*zitimanagementv1.CreateAppIdentityResponse, error) {
	appID, err := parseUUID(req.GetIdentityId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity_id: %v", err)
	}

	slug := strings.TrimSpace(req.GetSlug())
	if slug == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}

	zitiID, identityJSON, err := s.ziti.CreateAndEnrollAppIdentityWithOptions(ctx, appID, slug, req.GetAdditionalRoleAttributes(), req.GetTags())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create app identity: %v", err)
	}

	identity := store.ManagedIdentity{
		ZitiIdentityID: zitiID,
		IdentityID:     appID,
		IdentityType:   store.IdentityTypeApp,
	}
	if err := s.store.DeleteManagedIdentityByIdentityID(ctx, appID); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "ziti identity")
		return nil, status.Errorf(codes.Internal, "delete managed identity: %v", err)
	}
	if err := s.store.InsertManagedIdentity(ctx, identity); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "ziti identity")
		return nil, status.Errorf(codes.Internal, "insert managed identity: %v", err)
	}

	return &zitimanagementv1.CreateAppIdentityResponse{
		ZitiIdentityId: zitiID,
		IdentityJson:   identityJSON,
	}, nil
}

func (s *Server) CreateService(ctx context.Context, req *zitimanagementv1.CreateServiceRequest) (*zitimanagementv1.CreateServiceResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	roleAttributes := req.GetRoleAttributes()
	if len(roleAttributes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "role_attributes is required")
	}

	hostV1Config, err := fromProtoHostV1Config(req.GetHostV1Config())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "host_v1_config: %v", err)
	}
	interceptV1Config, err := fromProtoInterceptV1Config(req.GetInterceptV1Config())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "intercept_v1_config: %v", err)
	}

	var serviceID string
	if hostV1Config != nil || interceptV1Config != nil || len(req.GetTags()) > 0 {
		serviceID, err = s.ziti.CreateServiceWithConfigsAndTags(ctx, name, roleAttributes, hostV1Config, interceptV1Config, req.GetTags())
	} else {
		serviceID, err = s.ziti.CreateService(ctx, name, roleAttributes)
	}
	if err != nil {
		if req.GetReturnExisting() {
			service, getErr := s.getServiceByName(ctx, name)
			if getErr == nil {
				return &zitimanagementv1.CreateServiceResponse{ZitiServiceId: service.ID, ZitiServiceName: service.Name}, nil
			}
		}
		return nil, status.Errorf(codes.Internal, "create ziti service: %v", err)
	}

	return &zitimanagementv1.CreateServiceResponse{
		ZitiServiceId:   serviceID,
		ZitiServiceName: name,
	}, nil
}

func (s *Server) GetService(ctx context.Context, req *zitimanagementv1.GetServiceRequest) (*zitimanagementv1.GetServiceResponse, error) {
	serviceID := req.GetZitiServiceId()
	if serviceID != "" {
		service, err := s.ziti.GetService(ctx, serviceID)
		if err != nil {
			return nil, zitiResourceStatusError(err)
		}
		return &zitimanagementv1.GetServiceResponse{Service: toProtoOpenZitiService(service)}, nil
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_service_id or name is required")
	}
	service, err := s.getServiceByName(ctx, name)
	if err != nil {
		return nil, zitiResourceStatusError(err)
	}
	return &zitimanagementv1.GetServiceResponse{Service: toProtoOpenZitiService(service)}, nil
}

func (s *Server) ListServices(ctx context.Context, req *zitimanagementv1.ListServicesRequest) (*zitimanagementv1.ListServicesResponse, error) {
	result, err := s.ziti.ListServices(ctx, ziti.ServiceListFilter{
		Name:           strings.TrimSpace(req.GetName()),
		NamePrefix:     strings.TrimSpace(req.GetNamePrefix()),
		RoleAttributes: req.GetRoleAttributes(),
	}, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti services: %v", err)
	}
	resp := &zitimanagementv1.ListServicesResponse{Services: make([]*zitimanagementv1.OpenZitiService, len(result.Items)), NextPageToken: result.NextPageToken}
	for i, item := range result.Items {
		resp.Services[i] = toProtoOpenZitiService(item)
	}
	return resp, nil
}

func (s *Server) getServiceByName(ctx context.Context, name string) (ziti.OpenZitiService, error) {
	result, err := s.ziti.ListServices(ctx, ziti.ServiceListFilter{Name: name}, 2, "")
	if err != nil {
		return ziti.OpenZitiService{}, err
	}
	if len(result.Items) == 0 {
		return ziti.OpenZitiService{}, ziti.ErrServiceNotFound
	}
	return result.Items[0], nil
}

func (s *Server) CreateRunnerIdentity(ctx context.Context, req *zitimanagementv1.CreateRunnerIdentityRequest) (*zitimanagementv1.CreateRunnerIdentityResponse, error) {
	runnerID, err := parseUUID(req.GetRunnerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "runner_id: %v", err)
	}

	roleAttributes := req.GetRoleAttributes()
	if len(roleAttributes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "role_attributes is required")
	}

	zitiID, identityJSON, err := s.ziti.CreateAndEnrollRunnerIdentityWithTags(ctx, runnerID, roleAttributes, req.GetTags())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create runner identity: %v", err)
	}

	identity := store.ManagedIdentity{
		ZitiIdentityID: zitiID,
		IdentityID:     runnerID,
		IdentityType:   store.IdentityTypeRunner,
	}
	if err := s.store.DeleteManagedIdentityByIdentityID(ctx, runnerID); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "runner identity")
		return nil, status.Errorf(codes.Internal, "delete managed identity: %v", err)
	}
	if err := s.store.InsertManagedIdentity(ctx, identity); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "runner identity")
		return nil, status.Errorf(codes.Internal, "insert managed identity: %v", err)
	}

	return &zitimanagementv1.CreateRunnerIdentityResponse{
		ZitiIdentityId: zitiID,
		IdentityJson:   identityJSON,
	}, nil
}

func (s *Server) DeleteRunnerIdentity(ctx context.Context, req *zitimanagementv1.DeleteRunnerIdentityRequest) (*zitimanagementv1.DeleteRunnerIdentityResponse, error) {
	runnerID, err := parseUUID(req.GetIdentityId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity_id: %v", err)
	}

	identity, err := s.store.ResolveIdentityByIdentityID(ctx, runnerID)
	if err != nil {
		return nil, toStatusError(err)
	}

	if err := s.store.DeleteManagedIdentity(ctx, identity.ZitiIdentityID); err != nil {
		return nil, toStatusError(err)
	}

	if err := s.ziti.DeleteIdentity(ctx, identity.ZitiIdentityID); err != nil {
		if errors.Is(err, ziti.ErrIdentityNotFound) {
			log.Printf("ziti identity %s already deleted", identity.ZitiIdentityID)
		} else {
			return nil, status.Errorf(codes.Internal, "delete ziti identity: %v", err)
		}
	}

	zitiServiceID := req.GetZitiServiceId()
	if zitiServiceID == "" && identity.ZitiServiceID != nil {
		zitiServiceID = *identity.ZitiServiceID
	}
	if zitiServiceID != "" {
		if err := s.ziti.DeleteService(ctx, zitiServiceID); err != nil {
			if errors.Is(err, ziti.ErrServiceNotFound) {
				log.Printf("ziti service %s already deleted", zitiServiceID)
			} else {
				return nil, status.Errorf(codes.Internal, "delete ziti service: %v", err)
			}
		}
	}

	return &zitimanagementv1.DeleteRunnerIdentityResponse{}, nil
}

func (s *Server) DeleteIdentity(ctx context.Context, req *zitimanagementv1.DeleteIdentityRequest) (*zitimanagementv1.DeleteIdentityResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}
	if err := s.store.DeleteManagedIdentity(ctx, zitiID); err != nil {
		return nil, toStatusError(err)
	}
	if err := s.ziti.DeleteIdentity(ctx, zitiID); err != nil {
		if errors.Is(err, ziti.ErrIdentityNotFound) {
			log.Printf("ziti identity %s already deleted", zitiID)
			return &zitimanagementv1.DeleteIdentityResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "delete ziti identity: %v", err)
	}
	return &zitimanagementv1.DeleteIdentityResponse{}, nil
}

func (s *Server) DeleteAppIdentity(ctx context.Context, req *zitimanagementv1.DeleteAppIdentityRequest) (*zitimanagementv1.DeleteAppIdentityResponse, error) {
	appID, err := parseUUID(req.GetIdentityId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity_id: %v", err)
	}

	identity, resolveErr := s.store.ResolveIdentityByIdentityID(ctx, appID)
	identityFound := resolveErr == nil
	if resolveErr != nil && !errors.Is(resolveErr, store.ErrManagedIdentityNotFound) {
		return nil, toStatusError(resolveErr)
	}

	if identityFound {
		if err := s.store.DeleteManagedIdentity(ctx, identity.ZitiIdentityID); err != nil {
			if errors.Is(err, store.ErrManagedIdentityNotFound) {
				log.Printf("managed identity %s already deleted", identity.ZitiIdentityID)
			} else {
				return nil, toStatusError(err)
			}
		}

		if err := s.ziti.DeleteIdentity(ctx, identity.ZitiIdentityID); err != nil {
			if errors.Is(err, ziti.ErrIdentityNotFound) {
				log.Printf("ziti identity %s already deleted", identity.ZitiIdentityID)
			} else {
				return nil, status.Errorf(codes.Internal, "delete ziti identity: %v", err)
			}
		}
	}

	zitiServiceID := req.GetZitiServiceId()
	if zitiServiceID == "" && identityFound && identity.ZitiServiceID != nil {
		zitiServiceID = *identity.ZitiServiceID
	}
	if zitiServiceID == "" {
		if identityFound {
			return nil, status.Errorf(codes.Internal, "managed identity %s missing ziti service id", identity.ZitiIdentityID)
		}
		return nil, status.Error(codes.InvalidArgument, "ziti_service_id is required when managed identity is missing")
	}
	if err := s.ziti.DeleteService(ctx, zitiServiceID); err != nil {
		if errors.Is(err, ziti.ErrServiceNotFound) {
			log.Printf("ziti service %s already deleted", zitiServiceID)
		} else {
			return nil, status.Errorf(codes.Internal, "delete ziti service: %v", err)
		}
	}

	return &zitimanagementv1.DeleteAppIdentityResponse{}, nil
}

func (s *Server) RequestServiceIdentity(ctx context.Context, req *zitimanagementv1.RequestServiceIdentityRequest) (*zitimanagementv1.RequestServiceIdentityResponse, error) {
	serviceType, err := fromProtoServiceType(req.GetServiceType())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "service_type: %v", err)
	}

	name, roleAttributes, err := serviceIdentityConfig(serviceType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "service_type: %v", err)
	}

	zitiID, identityJSON, err := s.ziti.CreateAndEnrollServiceIdentity(ctx, name, roleAttributes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create service identity: %v", err)
	}

	leaseExpiresAt := time.Now().Add(s.serviceIdentityLeaseTTL)
	if err := s.store.InsertServiceIdentity(ctx, zitiID, serviceType, leaseExpiresAt); err != nil {
		s.cleanupZitiIdentity(ctx, zitiID, "service identity")
		return nil, status.Errorf(codes.Internal, "insert service identity: %v", err)
	}

	return &zitimanagementv1.RequestServiceIdentityResponse{
		ZitiIdentityId: zitiID,
		IdentityJson:   identityJSON,
	}, nil
}

func (s *Server) ExtendIdentityLease(ctx context.Context, req *zitimanagementv1.ExtendIdentityLeaseRequest) (*zitimanagementv1.ExtendIdentityLeaseResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}

	leaseExpiresAt := time.Now().Add(s.serviceIdentityLeaseTTL)
	if err := s.store.ExtendServiceIdentityLease(ctx, zitiID, leaseExpiresAt); err != nil {
		return nil, toStatusError(err)
	}
	return &zitimanagementv1.ExtendIdentityLeaseResponse{}, nil
}

func (s *Server) CreateServicePolicy(ctx context.Context, req *zitimanagementv1.CreateServicePolicyRequest) (*zitimanagementv1.CreateServicePolicyResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	policyType, err := fromProtoServicePolicyType(req.GetType())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "type: %v", err)
	}

	identityRoles := req.GetIdentityRoles()
	if len(identityRoles) == 0 {
		return nil, status.Error(codes.InvalidArgument, "identity_roles is required")
	}
	serviceRoles := req.GetServiceRoles()
	if len(serviceRoles) == 0 {
		return nil, status.Error(codes.InvalidArgument, "service_roles is required")
	}

	policyID, err := s.ziti.CreateServicePolicyWithTags(ctx, name, policyType, identityRoles, serviceRoles, req.GetTags())
	if err != nil {
		if req.GetReturnExisting() {
			policy, getErr := s.getServicePolicyByName(ctx, name)
			if getErr == nil {
				return &zitimanagementv1.CreateServicePolicyResponse{ZitiServicePolicyId: policy.ID}, nil
			}
		}
		return nil, status.Errorf(codes.Internal, "create ziti service policy: %v", err)
	}

	return &zitimanagementv1.CreateServicePolicyResponse{ZitiServicePolicyId: policyID}, nil
}

func (s *Server) GetServicePolicy(ctx context.Context, req *zitimanagementv1.GetServicePolicyRequest) (*zitimanagementv1.GetServicePolicyResponse, error) {
	policyID := req.GetZitiServicePolicyId()
	if policyID != "" {
		policy, err := s.ziti.GetServicePolicy(ctx, policyID)
		if err != nil {
			return nil, zitiResourceStatusError(err)
		}
		return &zitimanagementv1.GetServicePolicyResponse{ServicePolicy: toProtoOpenZitiServicePolicy(policy)}, nil
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_service_policy_id or name is required")
	}
	policy, err := s.getServicePolicyByName(ctx, name)
	if err != nil {
		return nil, zitiResourceStatusError(err)
	}
	return &zitimanagementv1.GetServicePolicyResponse{ServicePolicy: toProtoOpenZitiServicePolicy(policy)}, nil
}

func (s *Server) ListServicePolicies(ctx context.Context, req *zitimanagementv1.ListServicePoliciesRequest) (*zitimanagementv1.ListServicePoliciesResponse, error) {
	policyType, err := optionalProtoServicePolicyType(req.GetType())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "type: %v", err)
	}
	result, err := s.ziti.ListServicePolicies(ctx, ziti.ServicePolicyListFilter{
		Name:          strings.TrimSpace(req.GetName()),
		NamePrefix:    strings.TrimSpace(req.GetNamePrefix()),
		Type:          policyType,
		IdentityRoles: req.GetIdentityRoles(),
		ServiceRoles:  req.GetServiceRoles(),
	}, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti service policies: %v", err)
	}
	resp := &zitimanagementv1.ListServicePoliciesResponse{ServicePolicies: make([]*zitimanagementv1.OpenZitiServicePolicy, len(result.Items)), NextPageToken: result.NextPageToken}
	for i, item := range result.Items {
		resp.ServicePolicies[i] = toProtoOpenZitiServicePolicy(item)
	}
	return resp, nil
}

func (s *Server) getServicePolicyByName(ctx context.Context, name string) (ziti.OpenZitiServicePolicy, error) {
	result, err := s.ziti.ListServicePolicies(ctx, ziti.ServicePolicyListFilter{Name: name}, 2, "")
	if err != nil {
		return ziti.OpenZitiServicePolicy{}, err
	}
	if len(result.Items) == 0 {
		return ziti.OpenZitiServicePolicy{}, ziti.ErrServicePolicyNotFound
	}
	return result.Items[0], nil
}

func (s *Server) DeleteServicePolicy(ctx context.Context, req *zitimanagementv1.DeleteServicePolicyRequest) (*zitimanagementv1.DeleteServicePolicyResponse, error) {
	policyID := req.GetZitiServicePolicyId()
	if policyID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_service_policy_id is required")
	}

	if err := s.ziti.DeleteServicePolicy(ctx, policyID); err != nil {
		if errors.Is(err, ziti.ErrServicePolicyNotFound) {
			log.Printf("ziti service policy %s already deleted", policyID)
		} else {
			return nil, status.Errorf(codes.Internal, "delete ziti service policy: %v", err)
		}
	}

	return &zitimanagementv1.DeleteServicePolicyResponse{}, nil
}

func (s *Server) DeleteService(ctx context.Context, req *zitimanagementv1.DeleteServiceRequest) (*zitimanagementv1.DeleteServiceResponse, error) {
	serviceID := req.GetZitiServiceId()
	if serviceID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_service_id is required")
	}

	if err := s.ziti.DeleteService(ctx, serviceID); err != nil {
		if errors.Is(err, ziti.ErrServiceNotFound) {
			log.Printf("ziti service %s already deleted", serviceID)
		} else {
			return nil, status.Errorf(codes.Internal, "delete ziti service: %v", err)
		}
	}

	return &zitimanagementv1.DeleteServiceResponse{}, nil
}

func (s *Server) CreateDeviceIdentity(ctx context.Context, req *zitimanagementv1.CreateDeviceIdentityRequest) (*zitimanagementv1.CreateDeviceIdentityResponse, error) {
	userIdentityID, err := parseUUID(req.GetUserIdentityId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user_identity_id: %v", err)
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	zitiID, jwt, err := s.ziti.CreateDeviceIdentityWithOptions(ctx, userIdentityID, name, req.GetAdditionalRoleAttributes(), req.GetTags())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create device identity: %v", err)
	}

	return &zitimanagementv1.CreateDeviceIdentityResponse{
		ZitiIdentityId: zitiID,
		EnrollmentJwt:  jwt,
	}, nil
}

func (s *Server) DeleteDeviceIdentity(ctx context.Context, req *zitimanagementv1.DeleteDeviceIdentityRequest) (*zitimanagementv1.DeleteDeviceIdentityResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}

	if err := s.ziti.DeleteIdentity(ctx, zitiID); err != nil {
		if errors.Is(err, ziti.ErrIdentityNotFound) {
			log.Printf("ziti identity %s already deleted", zitiID)
		} else {
			return nil, status.Errorf(codes.Internal, "delete ziti identity: %v", err)
		}
	}

	return &zitimanagementv1.DeleteDeviceIdentityResponse{}, nil
}

func (s *Server) CreateTunnelIdentity(ctx context.Context, req *zitimanagementv1.CreateTunnelIdentityRequest) (*zitimanagementv1.CreateTunnelIdentityResponse, error) {
	networkID := strings.TrimSpace(req.GetNetworkId())
	if networkID == "" {
		return nil, status.Error(codes.InvalidArgument, "network_id is required")
	}
	tunnelCredentialID := strings.TrimSpace(req.GetTunnelCredentialId())
	if tunnelCredentialID == "" {
		return nil, status.Error(codes.InvalidArgument, "tunnel_credential_id is required")
	}

	zitiID, jwt, err := s.ziti.CreateTunnelIdentity(ctx, networkID, tunnelCredentialID, req.GetTags())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create tunnel identity: %v", err)
	}
	return &zitimanagementv1.CreateTunnelIdentityResponse{ZitiIdentityId: zitiID, EnrollmentJwt: jwt.Token, EnrollmentJwtExpiresAt: timestamppb.New(jwt.ExpiresAt)}, nil
}

func (s *Server) DeleteTunnelIdentity(ctx context.Context, req *zitimanagementv1.DeleteTunnelIdentityRequest) (*zitimanagementv1.DeleteTunnelIdentityResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}
	if err := s.ziti.DeleteIdentity(ctx, zitiID); err != nil && !errors.Is(err, ziti.ErrIdentityNotFound) {
		return nil, status.Errorf(codes.Internal, "delete ziti identity: %v", err)
	}
	return &zitimanagementv1.DeleteTunnelIdentityResponse{}, nil
}

func (s *Server) PatchIdentityRoleAttributes(ctx context.Context, req *zitimanagementv1.PatchIdentityRoleAttributesRequest) (*zitimanagementv1.PatchIdentityRoleAttributesResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}
	if err := s.ziti.PatchIdentityRoleAttributes(ctx, zitiID, req.GetAdd(), req.GetRemove()); err != nil {
		if errors.Is(err, ziti.ErrIdentityNotFound) {
			return nil, status.Error(codes.NotFound, "ziti identity not found")
		}
		return nil, status.Errorf(codes.Internal, "patch ziti identity role attributes: %v", err)
	}
	return &zitimanagementv1.PatchIdentityRoleAttributesResponse{}, nil
}

func (s *Server) GetIdentityLiveness(ctx context.Context, req *zitimanagementv1.GetIdentityLivenessRequest) (*zitimanagementv1.GetIdentityLivenessResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}
	liveness, err := s.ziti.GetIdentityLiveness(ctx, zitiID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get ziti identity liveness: %v", err)
	}
	enrollmentState := zitimanagementv1.IdentityEnrollmentState_IDENTITY_ENROLLMENT_STATE_ENROLLED
	if liveness.EnrollmentPending {
		enrollmentState = zitimanagementv1.IdentityEnrollmentState_IDENTITY_ENROLLMENT_STATE_PENDING
	}
	return &zitimanagementv1.GetIdentityLivenessResponse{EnrollmentState: enrollmentState, HasEdgeRouterConnection: liveness.HasEdgeRouterConnection}, nil
}

func (s *Server) ListServicesByTag(ctx context.Context, req *zitimanagementv1.ListServicesByTagRequest) (*zitimanagementv1.ListServicesByTagResponse, error) {
	result, err := s.ziti.ListServicesByTag(ctx, req.GetTags(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti services: %v", err)
	}
	resp := &zitimanagementv1.ListServicesByTagResponse{Services: make([]*zitimanagementv1.OpenZitiService, len(result.Items)), NextPageToken: result.NextPageToken}
	for i, item := range result.Items {
		resp.Services[i] = toProtoOpenZitiService(item)
	}
	return resp, nil
}

func (s *Server) ListIdentitiesByTag(ctx context.Context, req *zitimanagementv1.ListIdentitiesByTagRequest) (*zitimanagementv1.ListIdentitiesByTagResponse, error) {
	result, err := s.ziti.ListIdentitiesByTag(ctx, req.GetTags(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti identities: %v", err)
	}
	resp := &zitimanagementv1.ListIdentitiesByTagResponse{Identities: make([]*zitimanagementv1.OpenZitiIdentity, len(result.Items)), NextPageToken: result.NextPageToken}
	for i, item := range result.Items {
		resp.Identities[i] = toProtoOpenZitiIdentity(item)
	}
	return resp, nil
}

func (s *Server) ListServicePoliciesByTag(ctx context.Context, req *zitimanagementv1.ListServicePoliciesByTagRequest) (*zitimanagementv1.ListServicePoliciesByTagResponse, error) {
	result, err := s.ziti.ListServicePoliciesByTag(ctx, req.GetTags(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti service policies: %v", err)
	}
	resp := &zitimanagementv1.ListServicePoliciesByTagResponse{ServicePolicies: make([]*zitimanagementv1.OpenZitiServicePolicy, len(result.Items)), NextPageToken: result.NextPageToken}
	for i, item := range result.Items {
		resp.ServicePolicies[i] = toProtoOpenZitiServicePolicy(item)
	}
	return resp, nil
}

func (s *Server) UpdateService(ctx context.Context, req *zitimanagementv1.UpdateServiceRequest) (*zitimanagementv1.UpdateServiceResponse, error) {
	serviceID := req.GetZitiServiceId()
	if serviceID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_service_id is required")
	}
	hostV1Config, err := fromProtoHostV1Config(req.GetHostV1Config())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "host_v1_config: %v", err)
	}
	interceptV1Config, err := fromProtoInterceptV1Config(req.GetInterceptV1Config())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "intercept_v1_config: %v", err)
	}
	tags := req.GetTags()
	updateTags := len(tags) > 0
	if req.GetTagsUpdate() != nil {
		tags = req.GetTagsUpdate().GetTags()
		updateTags = true
	}
	updated, err := s.ziti.UpdateService(ctx, serviceID, hostV1Config, interceptV1Config, tags, updateTags)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update ziti service: %v", err)
	}
	return &zitimanagementv1.UpdateServiceResponse{Service: toProtoOpenZitiService(updated)}, nil
}

func (s *Server) ListManagedIdentities(ctx context.Context, req *zitimanagementv1.ListManagedIdentitiesRequest) (*zitimanagementv1.ListManagedIdentitiesResponse, error) {
	filter := store.ListFilter{}
	if req.GetIdentityType() != identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED {
		identityType, err := fromProtoIdentityType(req.GetIdentityType())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "identity_type: %v", err)
		}
		filter.IdentityType = &identityType
	}
	var cursor *store.PageCursor
	if token := req.GetPageToken(); token != "" {
		zitiID, err := store.DecodePageToken(token)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token: %v", err)
		}
		cursor = &store.PageCursor{AfterID: zitiID}
	}

	result, err := s.store.ListManagedIdentities(ctx, filter, req.GetPageSize(), cursor)
	if err != nil {
		return nil, toStatusError(err)
	}

	resp := &zitimanagementv1.ListManagedIdentitiesResponse{
		Identities: make([]*zitimanagementv1.ManagedIdentity, len(result.Identities)),
	}
	for i, identity := range result.Identities {
		protoIdentity, err := toProtoManagedIdentity(identity)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "managed identity: %v", err)
		}
		resp.Identities[i] = protoIdentity
	}
	if result.NextCursor != nil {
		pageToken, err := store.EncodePageToken(result.NextCursor.AfterID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode page token: %v", err)
		}
		resp.NextPageToken = pageToken
	}
	return resp, nil
}

func (s *Server) ResolveIdentity(ctx context.Context, req *zitimanagementv1.ResolveIdentityRequest) (*zitimanagementv1.ResolveIdentityResponse, error) {
	zitiID := req.GetZitiIdentityId()
	if zitiID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_identity_id is required")
	}
	identity, err := s.store.ResolveIdentity(ctx, zitiID)
	if err != nil {
		if s.resolveIdentityName && errors.Is(err, store.ErrManagedIdentityNotFound) {
			agentID, ok := parseAgentIdentityName(zitiID)
			if ok {
				identity, err = s.store.ResolveIdentityByIdentityID(ctx, agentID)
				if err != nil {
					return nil, toStatusError(err)
				}
				return resolveIdentityResponse(identity)
			}
		}
		return nil, toStatusError(err)
	}
	return resolveIdentityResponse(identity)
}

const agentIdentityNamePrefix = "agent-"
const agentIdentityUUIDLength = 36

func parseAgentIdentityName(value string) (uuid.UUID, bool) {
	if !strings.HasPrefix(value, agentIdentityNamePrefix) {
		return uuid.UUID{}, false
	}
	trimmed := value[len(agentIdentityNamePrefix):]
	if len(trimmed) < agentIdentityUUIDLength {
		return uuid.UUID{}, false
	}
	candidate := trimmed[:agentIdentityUUIDLength]
	if len(trimmed) > agentIdentityUUIDLength && trimmed[agentIdentityUUIDLength] != '-' {
		return uuid.UUID{}, false
	}
	agentID, err := uuid.Parse(candidate)
	if err != nil {
		return uuid.UUID{}, false
	}
	return agentID, true
}

func resolveIdentityResponse(identity store.ManagedIdentity) (*zitimanagementv1.ResolveIdentityResponse, error) {
	identityType, err := toProtoIdentityType(identity.IdentityType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "identity_type: %v", err)
	}
	resp := &zitimanagementv1.ResolveIdentityResponse{
		IdentityId:   identity.IdentityID.String(),
		IdentityType: identityType,
	}
	if identity.WorkloadID != nil {
		workloadID := identity.WorkloadID.String()
		resp.WorkloadId = &workloadID
	}
	return resp, nil
}

func toStatusError(err error) error {
	switch {
	case errors.Is(err, store.ErrManagedIdentityNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, store.ErrServiceIdentityNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

func zitiResourceStatusError(err error) error {
	switch {
	case errors.Is(err, ziti.ErrServiceNotFound), errors.Is(err, ziti.ErrServicePolicyNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

func serviceIdentityConfig(serviceType store.ServiceType) (string, []string, error) {
	suffix := id.ShortUUID()
	switch serviceType {
	case store.ServiceTypeGateway:
		return fmt.Sprintf("svc-gateway-%s", suffix), []string{"gateway-hosts"}, nil
	case store.ServiceTypeOrchestrator:
		return fmt.Sprintf("svc-orchestrator-%s", suffix), []string{"orchestrators"}, nil
	case store.ServiceTypeLLMProxy:
		return fmt.Sprintf("svc-llm-proxy-%s", suffix), []string{"llm-proxy-hosts"}, nil
	case store.ServiceTypeTracing:
		return fmt.Sprintf("svc-tracing-%s", suffix), []string{"tracing-hosts"}, nil
	case store.ServiceTypeRunners:
		return fmt.Sprintf("svc-runners-%s", suffix), []string{"runners-service-hosts"}, nil
	case store.ServiceTypeEgressGateway:
		return fmt.Sprintf("svc-egress-gateway-%s", suffix), []string{"egress-gateway-hosts"}, nil
	case store.ServiceTypeTerminalProxy:
		// Dials runner-{runnerId} to open exec streams; the static
		// terminal-proxy-dial-runners policy grants this attribute against
		// #runner-services.
		return fmt.Sprintf("svc-terminal-proxy-%s", suffix), []string{"terminal-proxy-hosts"}, nil
	case store.ServiceTypeUnspecified:
		return "", nil, fmt.Errorf("service type unspecified")
	default:
		return "", nil, fmt.Errorf("unknown service type %d", serviceType)
	}
}
