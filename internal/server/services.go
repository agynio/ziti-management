package server

import (
	"context"
	"errors"
	"strings"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/ziti"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetService(ctx context.Context, req *zitimanagementv1.GetServiceRequest) (*zitimanagementv1.GetServiceResponse, error) {
	service, err := s.getService(ctx, req)
	if err != nil {
		return nil, err
	}
	return &zitimanagementv1.GetServiceResponse{Service: toProtoService(service)}, nil
}

func (s *Server) ListServices(ctx context.Context, req *zitimanagementv1.ListServicesRequest) (*zitimanagementv1.ListServicesResponse, error) {
	name := strings.TrimSpace(req.GetName())
	namePrefix := strings.TrimSpace(req.GetNamePrefix())
	if name != "" && namePrefix != "" {
		return nil, status.Error(codes.InvalidArgument, "only one of name or name_prefix may be set")
	}
	result, err := s.ziti.ListServices(ctx, ziti.ServiceListFilter{
		Name:           name,
		NamePrefix:     namePrefix,
		RoleAttributes: cleanStringList(req.GetRoleAttributes()),
		PageSize:       req.GetPageSize(),
		PageToken:      req.GetPageToken(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti services: %v", err)
	}
	services := make([]*zitimanagementv1.Service, 0, len(result.Services))
	for _, service := range result.Services {
		services = append(services, toProtoService(service))
	}
	return &zitimanagementv1.ListServicesResponse{Services: services, NextPageToken: result.NextPageToken}, nil
}

func (s *Server) UpdateService(ctx context.Context, req *zitimanagementv1.UpdateServiceRequest) (*zitimanagementv1.UpdateServiceResponse, error) {
	serviceID := strings.TrimSpace(req.GetZitiServiceId())
	if serviceID == "" {
		return nil, status.Error(codes.InvalidArgument, "ziti_service_id is required")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	roleAttributes := cleanStringList(req.GetRoleAttributes())
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
	service, err := s.ziti.UpdateService(ctx, serviceID, name, roleAttributes, hostV1Config, interceptV1Config)
	if err != nil {
		if errors.Is(err, ziti.ErrServiceNotFound) {
			return nil, status.Error(codes.NotFound, "ziti service not found")
		}
		return nil, status.Errorf(codes.Internal, "update ziti service: %v", err)
	}
	return &zitimanagementv1.UpdateServiceResponse{
		Service:           &zitimanagementv1.OpenZitiService{ZitiServiceId: service.ID, Name: service.Name, RoleAttributes: service.RoleAttributes},
		ReconciledService: toProtoService(service),
	}, nil
}

func (s *Server) GetServicePolicy(ctx context.Context, req *zitimanagementv1.GetServicePolicyRequest) (*zitimanagementv1.GetServicePolicyResponse, error) {
	policy, err := s.getServicePolicy(ctx, req)
	if err != nil {
		return nil, err
	}
	protoPolicy, err := toProtoServicePolicy(policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert ziti service policy: %v", err)
	}
	return &zitimanagementv1.GetServicePolicyResponse{ServicePolicy: protoPolicy}, nil
}

func (s *Server) ListServicePolicies(ctx context.Context, req *zitimanagementv1.ListServicePoliciesRequest) (*zitimanagementv1.ListServicePoliciesResponse, error) {
	name := strings.TrimSpace(req.GetName())
	namePrefix := strings.TrimSpace(req.GetNamePrefix())
	if name != "" && namePrefix != "" {
		return nil, status.Error(codes.InvalidArgument, "only one of name or name_prefix may be set")
	}
	var policyType string
	var err error
	if req.GetType() != zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_UNSPECIFIED {
		policyType, err = fromProtoServicePolicyType(req.GetType())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "type: %v", err)
		}
	}
	result, err := s.ziti.ListServicePolicies(ctx, ziti.ServicePolicyListFilter{
		Name:          name,
		NamePrefix:    namePrefix,
		Type:          policyType,
		IdentityRoles: cleanStringList(req.GetIdentityRoles()),
		ServiceRoles:  cleanStringList(req.GetServiceRoles()),
		PageSize:      req.GetPageSize(),
		PageToken:     req.GetPageToken(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ziti service policies: %v", err)
	}
	policies := make([]*zitimanagementv1.ServicePolicy, 0, len(result.ServicePolicies))
	for _, policy := range result.ServicePolicies {
		protoPolicy, err := toProtoServicePolicy(policy)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert ziti service policy: %v", err)
		}
		policies = append(policies, protoPolicy)
	}
	return &zitimanagementv1.ListServicePoliciesResponse{ServicePolicies: policies, NextPageToken: result.NextPageToken}, nil
}

func (s *Server) getService(ctx context.Context, req *zitimanagementv1.GetServiceRequest) (ziti.Service, error) {
	serviceID := strings.TrimSpace(req.GetZitiServiceId())
	name := strings.TrimSpace(req.GetName())
	if serviceID == "" && name == "" {
		return ziti.Service{}, status.Error(codes.InvalidArgument, "ziti_service_id or name is required")
	}
	if serviceID != "" && name != "" {
		return ziti.Service{}, status.Error(codes.InvalidArgument, "only one of ziti_service_id or name may be set")
	}
	var service ziti.Service
	var err error
	if serviceID != "" {
		service, err = s.ziti.GetService(ctx, serviceID)
	} else {
		service, err = s.ziti.GetServiceByName(ctx, name)
	}
	if err != nil {
		if errors.Is(err, ziti.ErrServiceNotFound) {
			return ziti.Service{}, status.Error(codes.NotFound, "ziti service not found")
		}
		return ziti.Service{}, status.Errorf(codes.Internal, "get ziti service: %v", err)
	}
	return service, nil
}

func (s *Server) getServicePolicy(ctx context.Context, req *zitimanagementv1.GetServicePolicyRequest) (ziti.ServicePolicy, error) {
	policyID := strings.TrimSpace(req.GetZitiServicePolicyId())
	name := strings.TrimSpace(req.GetName())
	if policyID == "" && name == "" {
		return ziti.ServicePolicy{}, status.Error(codes.InvalidArgument, "ziti_service_policy_id or name is required")
	}
	if policyID != "" && name != "" {
		return ziti.ServicePolicy{}, status.Error(codes.InvalidArgument, "only one of ziti_service_policy_id or name may be set")
	}
	var policy ziti.ServicePolicy
	var err error
	if policyID != "" {
		policy, err = s.ziti.GetServicePolicy(ctx, policyID)
	} else {
		policy, err = s.ziti.GetServicePolicyByName(ctx, name)
	}
	if err != nil {
		if errors.Is(err, ziti.ErrServicePolicyNotFound) {
			return ziti.ServicePolicy{}, status.Error(codes.NotFound, "ziti service policy not found")
		}
		return ziti.ServicePolicy{}, status.Errorf(codes.Internal, "get ziti service policy: %v", err)
	}
	return policy, nil
}

func cleanStringList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
