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
	CreateAndEnrollAppIdentity(ctx context.Context, appID uuid.UUID, slug string) (string, []byte, error)
	CreateAndEnrollRunnerIdentity(ctx context.Context, runnerID uuid.UUID, roleAttributes []string) (string, []byte, error)
	CreateAndEnrollServiceIdentity(ctx context.Context, name string, roleAttributes []string) (string, []byte, error)
	CreateService(ctx context.Context, name string, roleAttributes []string) (string, error)
	CreateServiceWithConfigs(ctx context.Context, name string, roleAttributes []string, hostV1 *ziti.HostV1ConfigData, interceptV1 *ziti.InterceptV1ConfigData) (string, error)
	CreateServicePolicy(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string) (string, error)
	CreateDeviceIdentity(ctx context.Context, userIdentityID uuid.UUID, name string) (string, string, error)
	DeleteIdentity(ctx context.Context, zitiIdentityID string) error
	DeleteService(ctx context.Context, serviceID string) error
	DeleteServicePolicy(ctx context.Context, policyID string) error
}

type Server struct {
	zitimanagementv1.UnimplementedZitiManagementServiceServer
	store                   managedIdentityStore
	ziti                    zitiClient
	serviceIdentityLeaseTTL time.Duration
}

func (s *Server) cleanupZitiIdentity(ctx context.Context, zitiID, label string) {
	if err := s.ziti.DeleteIdentity(ctx, zitiID); err != nil && !errors.Is(err, ziti.ErrIdentityNotFound) {
		log.Printf("failed to cleanup %s %s: %v", label, zitiID, err)
	}
}

func New(store managedIdentityStore, zitiClient zitiClient, serviceIdentityLeaseTTL time.Duration) *Server {
	return &Server{store: store, ziti: zitiClient, serviceIdentityLeaseTTL: serviceIdentityLeaseTTL}
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

	zitiID, jwt, err := s.ziti.CreateAgentIdentity(ctx, agentID, workloadID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create ziti identity: %v", err)
	}

	identity := store.ManagedIdentity{
		ZitiIdentityID: zitiID,
		IdentityID:     agentID,
		IdentityType:   store.IdentityTypeAgent,
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

func (s *Server) CreateAppIdentity(ctx context.Context, req *zitimanagementv1.CreateAppIdentityRequest) (*zitimanagementv1.CreateAppIdentityResponse, error) {
	appID, err := parseUUID(req.GetIdentityId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity_id: %v", err)
	}

	slug := strings.TrimSpace(req.GetSlug())
	if slug == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}

	zitiID, identityJSON, err := s.ziti.CreateAndEnrollAppIdentity(ctx, appID, slug)
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
	if hostV1Config != nil || interceptV1Config != nil {
		serviceID, err = s.ziti.CreateServiceWithConfigs(ctx, name, roleAttributes, hostV1Config, interceptV1Config)
	} else {
		serviceID, err = s.ziti.CreateService(ctx, name, roleAttributes)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create ziti service: %v", err)
	}

	return &zitimanagementv1.CreateServiceResponse{
		ZitiServiceId:   serviceID,
		ZitiServiceName: name,
	}, nil
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

	zitiID, identityJSON, err := s.ziti.CreateAndEnrollRunnerIdentity(ctx, runnerID, roleAttributes)
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

	identity, err := s.store.ResolveIdentityByIdentityID(ctx, appID)
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
	if zitiServiceID == "" {
		return nil, status.Errorf(codes.Internal, "managed identity %s missing ziti service id", identity.ZitiIdentityID)
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

	policyID, err := s.ziti.CreateServicePolicy(ctx, name, policyType, identityRoles, serviceRoles)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create ziti service policy: %v", err)
	}

	return &zitimanagementv1.CreateServicePolicyResponse{ZitiServicePolicyId: policyID}, nil
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

	zitiID, jwt, err := s.ziti.CreateDeviceIdentity(ctx, userIdentityID, name)
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
		return nil, toStatusError(err)
	}
	identityType, err := toProtoIdentityType(identity.IdentityType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "identity_type: %v", err)
	}
	return &zitimanagementv1.ResolveIdentityResponse{
		IdentityId:   identity.IdentityID.String(),
		IdentityType: identityType,
	}, nil
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

func serviceIdentityConfig(serviceType store.ServiceType) (string, []string, error) {
	suffix := id.ShortUUID()
	switch serviceType {
	case store.ServiceTypeGateway:
		return fmt.Sprintf("svc-gateway-%s", suffix), []string{"gateway-hosts"}, nil
	case store.ServiceTypeOrchestrator:
		return fmt.Sprintf("svc-orchestrator-%s", suffix), []string{"orchestrators"}, nil
	case store.ServiceTypeLLMProxy:
		return fmt.Sprintf("svc-llm-proxy-%s", suffix), []string{"llm-proxy-hosts"}, nil
	case store.ServiceTypeUnspecified:
		return "", nil, fmt.Errorf("service type unspecified")
	default:
		return "", nil, fmt.Errorf("unknown service type %d", serviceType)
	}
}
