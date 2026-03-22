package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
)

type Server struct {
	zitimanagementv1.UnimplementedZitiManagementServiceServer
	store                   *store.Store
	ziti                    *ziti.Client
	serviceIdentityLeaseTTL time.Duration
}

func New(store *store.Store, zitiClient *ziti.Client, serviceIdentityLeaseTTL time.Duration) *Server {
	return &Server{store: store, ziti: zitiClient, serviceIdentityLeaseTTL: serviceIdentityLeaseTTL}
}

func (s *Server) CreateAgentIdentity(ctx context.Context, req *zitimanagementv1.CreateAgentIdentityRequest) (*zitimanagementv1.CreateAgentIdentityResponse, error) {
	agentID, err := parseUUID(req.GetAgentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "agent_id: %v", err)
	}

	zitiID, jwt, err := s.ziti.CreateAgentIdentity(ctx, agentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create ziti identity: %v", err)
	}

	identity := store.ManagedIdentity{
		ZitiIdentityID: zitiID,
		IdentityID:     agentID,
		IdentityType:   store.IdentityTypeAgent,
		TenantID:       uuid.Nil,
	}
	if err := s.store.InsertManagedIdentity(ctx, identity); err != nil {
		cleanupErr := s.ziti.DeleteIdentity(ctx, zitiID)
		if cleanupErr != nil && !errors.Is(cleanupErr, ziti.ErrIdentityNotFound) {
			log.Printf("failed to cleanup ziti identity %s: %v", zitiID, cleanupErr)
		}
		return nil, status.Errorf(codes.Internal, "insert managed identity: %v", err)
	}

	return &zitimanagementv1.CreateAgentIdentityResponse{
		ZitiIdentityId: zitiID,
		EnrollmentJwt:  jwt,
	}, nil
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
		cleanupErr := s.ziti.DeleteIdentity(ctx, zitiID)
		if cleanupErr != nil && !errors.Is(cleanupErr, ziti.ErrIdentityNotFound) {
			log.Printf("failed to cleanup ziti identity %s: %v", zitiID, cleanupErr)
		}
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

func (s *Server) ListManagedIdentities(ctx context.Context, req *zitimanagementv1.ListManagedIdentitiesRequest) (*zitimanagementv1.ListManagedIdentitiesResponse, error) {
	filter := store.ListFilter{}
	if req.GetIdentityType() != zitimanagementv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED {
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
	suffix := shortUUID()
	switch serviceType {
	case store.ServiceTypeGateway:
		return fmt.Sprintf("svc-gateway-%s", suffix), []string{"gateway-hosts"}, nil
	case store.ServiceTypeOrchestrator:
		return fmt.Sprintf("svc-orchestrator-%s", suffix), []string{"orchestrators"}, nil
	case store.ServiceTypeRunner:
		return fmt.Sprintf("svc-runner-%s", suffix), []string{"runners"}, nil
	case store.ServiceTypeUnspecified:
		return "", nil, fmt.Errorf("service type unspecified")
	default:
		return "", nil, fmt.Errorf("unknown service type %d", serviceType)
	}
}

func shortUUID() string {
	id := uuid.NewString()
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
