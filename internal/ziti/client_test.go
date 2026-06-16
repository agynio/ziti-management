package ziti

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_model"
	sdkziti "github.com/openziti/sdk-golang/ziti"
)

type fakeIdentityService struct {
	createIdentityFunc func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error)
	deleteIdentityFunc func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error)
	detailIdentityFunc func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error)
	listIdentitiesFunc func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error)
	patchIdentityFunc  func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error)
}

func (f *fakeIdentityService) CreateIdentity(params *identity.CreateIdentityParams, _ runtime.ClientAuthInfoWriter, _ ...identity.ClientOption) (*identity.CreateIdentityCreated, error) {
	if f.createIdentityFunc == nil {
		return nil, errors.New("create identity not stubbed")
	}
	return f.createIdentityFunc(params)
}

func (f *fakeIdentityService) DeleteIdentity(params *identity.DeleteIdentityParams, _ runtime.ClientAuthInfoWriter, _ ...identity.ClientOption) (*identity.DeleteIdentityOK, error) {
	if f.deleteIdentityFunc == nil {
		return nil, errors.New("delete identity not stubbed")
	}
	return f.deleteIdentityFunc(params)
}

func (f *fakeIdentityService) DetailIdentity(params *identity.DetailIdentityParams, _ runtime.ClientAuthInfoWriter, _ ...identity.ClientOption) (*identity.DetailIdentityOK, error) {
	if f.detailIdentityFunc == nil {
		return nil, errors.New("detail identity not stubbed")
	}
	return f.detailIdentityFunc(params)
}

func (f *fakeIdentityService) ListIdentities(params *identity.ListIdentitiesParams, _ runtime.ClientAuthInfoWriter, _ ...identity.ClientOption) (*identity.ListIdentitiesOK, error) {
	if f.listIdentitiesFunc == nil {
		return nil, errors.New("list identities not stubbed")
	}
	return f.listIdentitiesFunc(params)
}

func (f *fakeIdentityService) PatchIdentity(params *identity.PatchIdentityParams, _ runtime.ClientAuthInfoWriter, _ ...identity.ClientOption) (*identity.PatchIdentityOK, error) {
	if f.patchIdentityFunc == nil {
		return nil, errors.New("patch identity not stubbed")
	}
	return f.patchIdentityFunc(params)
}

type fakeServiceService struct {
	createServiceFunc func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error)
	deleteServiceFunc func(params *service.DeleteServiceParams) (*service.DeleteServiceOK, error)
	detailServiceFunc func(params *service.DetailServiceParams) (*service.DetailServiceOK, error)
	listConfigFunc    func(params *service.ListServiceConfigParams) (*service.ListServiceConfigOK, error)
	listServicesFunc  func(params *service.ListServicesParams) (*service.ListServicesOK, error)
	patchServiceFunc  func(params *service.PatchServiceParams) (*service.PatchServiceOK, error)
}

func (f *fakeServiceService) CreateService(params *service.CreateServiceParams, _ runtime.ClientAuthInfoWriter, _ ...service.ClientOption) (*service.CreateServiceCreated, error) {
	if f.createServiceFunc == nil {
		return nil, errors.New("create service not stubbed")
	}
	return f.createServiceFunc(params)
}

func (f *fakeServiceService) DeleteService(params *service.DeleteServiceParams, _ runtime.ClientAuthInfoWriter, _ ...service.ClientOption) (*service.DeleteServiceOK, error) {
	if f.deleteServiceFunc == nil {
		return nil, errors.New("delete service not stubbed")
	}
	return f.deleteServiceFunc(params)
}

func (f *fakeServiceService) DetailService(params *service.DetailServiceParams, _ runtime.ClientAuthInfoWriter, _ ...service.ClientOption) (*service.DetailServiceOK, error) {
	if f.detailServiceFunc == nil {
		return nil, errors.New("detail service not stubbed")
	}
	return f.detailServiceFunc(params)
}

func (f *fakeServiceService) ListServiceConfig(params *service.ListServiceConfigParams, _ runtime.ClientAuthInfoWriter, _ ...service.ClientOption) (*service.ListServiceConfigOK, error) {
	if f.listConfigFunc == nil {
		return nil, errors.New("list service config not stubbed")
	}
	return f.listConfigFunc(params)
}

func (f *fakeServiceService) ListServices(params *service.ListServicesParams, _ runtime.ClientAuthInfoWriter, _ ...service.ClientOption) (*service.ListServicesOK, error) {
	if f.listServicesFunc == nil {
		return nil, errors.New("list services not stubbed")
	}
	return f.listServicesFunc(params)
}

func (f *fakeServiceService) PatchService(params *service.PatchServiceParams, _ runtime.ClientAuthInfoWriter, _ ...service.ClientOption) (*service.PatchServiceOK, error) {
	if f.patchServiceFunc == nil {
		return nil, errors.New("patch service not stubbed")
	}
	return f.patchServiceFunc(params)
}

type fakeConfigService struct {
	createConfigFunc func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error)
	deleteConfigFunc func(params *config.DeleteConfigParams) (*config.DeleteConfigOK, error)
	patchConfigFunc  func(params *config.PatchConfigParams) (*config.PatchConfigOK, error)
}

func (f *fakeConfigService) CreateConfig(params *config.CreateConfigParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.CreateConfigCreated, error) {
	if f.createConfigFunc == nil {
		return nil, errors.New("create config not stubbed")
	}
	return f.createConfigFunc(params)
}

func (f *fakeConfigService) DeleteConfig(params *config.DeleteConfigParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.DeleteConfigOK, error) {
	if f.deleteConfigFunc == nil {
		return nil, errors.New("delete config not stubbed")
	}
	return f.deleteConfigFunc(params)
}

func (f *fakeConfigService) PatchConfig(params *config.PatchConfigParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.PatchConfigOK, error) {
	if f.patchConfigFunc == nil {
		return nil, errors.New("patch config not stubbed")
	}
	return f.patchConfigFunc(params)
}

type fakeServicePolicyService struct {
	createServicePolicyFunc func(params *service_policy.CreateServicePolicyParams) (*service_policy.CreateServicePolicyCreated, error)
	deleteServicePolicyFunc func(params *service_policy.DeleteServicePolicyParams) (*service_policy.DeleteServicePolicyOK, error)
	detailServicePolicyFunc func(params *service_policy.DetailServicePolicyParams) (*service_policy.DetailServicePolicyOK, error)
	listServicePoliciesFunc func(params *service_policy.ListServicePoliciesParams) (*service_policy.ListServicePoliciesOK, error)
}

func (f *fakeServicePolicyService) CreateServicePolicy(params *service_policy.CreateServicePolicyParams, _ runtime.ClientAuthInfoWriter, _ ...service_policy.ClientOption) (*service_policy.CreateServicePolicyCreated, error) {
	if f.createServicePolicyFunc == nil {
		return nil, errors.New("create service policy not stubbed")
	}
	return f.createServicePolicyFunc(params)
}

func (f *fakeServicePolicyService) DeleteServicePolicy(params *service_policy.DeleteServicePolicyParams, _ runtime.ClientAuthInfoWriter, _ ...service_policy.ClientOption) (*service_policy.DeleteServicePolicyOK, error) {
	if f.deleteServicePolicyFunc == nil {
		return nil, errors.New("delete service policy not stubbed")
	}
	return f.deleteServicePolicyFunc(params)
}

func (f *fakeServicePolicyService) DetailServicePolicy(params *service_policy.DetailServicePolicyParams, _ runtime.ClientAuthInfoWriter, _ ...service_policy.ClientOption) (*service_policy.DetailServicePolicyOK, error) {
	if f.detailServicePolicyFunc == nil {
		return nil, errors.New("detail service policy not stubbed")
	}
	return f.detailServicePolicyFunc(params)
}

func (f *fakeServicePolicyService) ListServicePolicies(params *service_policy.ListServicePoliciesParams, _ runtime.ClientAuthInfoWriter, _ ...service_policy.ClientOption) (*service_policy.ListServicePoliciesOK, error) {
	if f.listServicePoliciesFunc == nil {
		return nil, errors.New("list service policies not stubbed")
	}
	return f.listServicePoliciesFunc(params)
}

func TestCreateAgentIdentityCreatesIdentity(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	workloadID := uuid.New()
	createdID := "created-id"
	jwt := "jwt-token"

	fake := &fakeIdentityService{
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			t.Fatalf("delete identity should not be called: %#v", params)
			return nil, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, workloadID)
			assertCreateAgentRoleAttributes(t, params, agentID, workloadID)
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			if params == nil || params.ID != createdID {
				t.Fatalf("expected detail identity id %q, got %#v", createdID, params)
			}
			return detailIdentityResponse(jwt), nil
		},
	}

	client := &Client{identity: fake}
	zitiID, token, err := client.CreateAgentIdentity(ctx, agentID, workloadID)
	if err != nil {
		t.Fatalf("create agent identity: %v", err)
	}
	if zitiID != createdID {
		t.Fatalf("expected identity id %q, got %q", createdID, zitiID)
	}
	if token != jwt {
		t.Fatalf("expected jwt %q, got %q", jwt, token)
	}
}

func TestCreateAgentIdentityWithOptionsAddsRolesAndTags(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	workloadID := uuid.New()
	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateAgentRoleAttributes(t, params, agentID, workloadID, "group-one")
			assertTags(t, params.Identity.Tags, map[string]string{"network": "net-1"})
			return createIdentityResponse("identity-id"), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return detailIdentityResponse("jwt-token"), nil
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateAgentIdentityWithOptions(ctx, agentID, workloadID, []string{"group-one"}, map[string]string{"network": "net-1"})
	if err != nil {
		t.Fatalf("create agent identity: %v", err)
	}
}

func TestCreateTunnelIdentityCreatesTunnelRolesAndTags(t *testing.T) {
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			if params == nil || params.Identity == nil || params.Identity.RoleAttributes == nil {
				t.Fatalf("expected role attributes")
			}
			expectedRoles := rest_model.Attributes{"tunnels", "network-network-1"}
			if !reflect.DeepEqual(*params.Identity.RoleAttributes, expectedRoles) {
				t.Fatalf("unexpected role attributes: %#v", params.Identity.RoleAttributes)
			}
			assertTags(t, params.Identity.Tags, map[string]string{"owner": "networks"})
			return createIdentityResponse("tunnel-id"), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return detailIdentityResponseWithExpiry("tunnel-jwt", expiresAt), nil
		},
	}

	client := &Client{identity: fake}
	zitiID, jwt, err := client.CreateTunnelIdentity(ctx, "network-1", "credential-1", map[string]string{"owner": "networks"})
	if err != nil {
		t.Fatalf("create tunnel identity: %v", err)
	}
	if zitiID != "tunnel-id" || jwt.Token != "tunnel-jwt" || !jwt.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("unexpected create result %q %q", zitiID, jwt)
	}
}

func TestCreateTunnelIdentityUsesJWTExpirationWhenControllerExpiryMissing(t *testing.T) {
	ctx := context.Background()
	expiresAt := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	stubEnrollmentFuncs(t, func(token string) (*sdkziti.EnrollmentClaims, *jwt.Token, error) {
		if token != "tunnel-jwt" {
			t.Fatalf("unexpected token: %s", token)
		}
		return &sdkziti.EnrollmentClaims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt)}}, nil, nil
	}, nil)
	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			return createIdentityResponse("tunnel-id"), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return &identity.DetailIdentityOK{Payload: &rest_model.DetailIdentityEnvelope{Data: &rest_model.IdentityDetail{Enrollment: &rest_model.IdentityEnrollments{
				Ott: &rest_model.IdentityEnrollmentsOtt{JWT: "tunnel-jwt"},
			}}}}, nil
		},
	}

	client := &Client{identity: fake}
	_, jwt, err := client.CreateTunnelIdentity(ctx, "network-1", "credential-1", nil)
	if err != nil {
		t.Fatalf("create tunnel identity: %v", err)
	}
	if !jwt.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expiry %s, got %s", expiresAt, jwt.ExpiresAt)
	}
}

func TestPatchIdentityRoleAttributesAddsAttrs(t *testing.T) {
	ctx := context.Background()
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(rest_model.Attributes{"agents", "agent-one"}),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			assertPatchIdentityRoleAttributes(t, params, "identity-id", []string{"agents", "agent-one", "group-one", "group-two"})
			return &identity.PatchIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	if err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-one", "group-two"}, nil); err != nil {
		t.Fatalf("patch identity role attributes: %v", err)
	}
}

func TestPatchIdentityRoleAttributesRemovesAttrs(t *testing.T) {
	ctx := context.Background()
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(rest_model.Attributes{"agents", "group-one", "agent-one", "group-two"}),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			assertPatchIdentityRoleAttributes(t, params, "identity-id", []string{"agents", "agent-one"})
			return &identity.PatchIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	if err := client.PatchIdentityRoleAttributes(ctx, "identity-id", nil, []string{"group-one", "group-two"}); err != nil {
		t.Fatalf("patch identity role attributes: %v", err)
	}
}

func TestPatchIdentityRoleAttributesAddsAndRemovesDeterministically(t *testing.T) {
	ctx := context.Background()
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(rest_model.Attributes{"agents", "group-old", "agent-one", "group-same"}),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			assertPatchIdentityRoleAttributes(t, params, "identity-id", []string{"agents", "agent-one", "group-new"})
			return &identity.PatchIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-new", "group-same"}, []string{"group-old", "group-same"})
	if err != nil {
		t.Fatalf("patch identity role attributes: %v", err)
	}
}

func TestPatchIdentityRoleAttributesIsIdempotent(t *testing.T) {
	ctx := context.Background()
	currentAttrs := rest_model.Attributes{"agents", "group-one", "agent-one"}
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(currentAttrs),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			assertPatchIdentityRoleAttributes(t, params, "identity-id", []string{"agents", "group-one", "agent-one"})
			return &identity.PatchIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-one", "group-one"}, []string{"missing", "missing"})
	if err != nil {
		t.Fatalf("patch identity role attributes: %v", err)
	}
}

func TestPatchIdentityRoleAttributesPreservesUnrelatedAttrs(t *testing.T) {
	ctx := context.Background()
	currentAttrs := rest_model.Attributes{"agents", "agent-one", "workload-one", "devices", "user-one", "apps", "app-one", "custom"}
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(currentAttrs),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			assertPatchIdentityRoleAttributes(t, params, "identity-id", []string{"agents", "agent-one", "workload-one", "devices", "user-one", "apps", "app-one", "custom", "group-one"})
			return &identity.PatchIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	if err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-one"}, []string{"missing"}); err != nil {
		t.Fatalf("patch identity role attributes: %v", err)
	}
}

func TestPatchIdentityRoleAttributesMapsDetailNotFound(t *testing.T) {
	ctx := context.Background()
	fake := &fakeIdentityService{
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return nil, &identity.DetailIdentityNotFound{}
		},
	}

	client := &Client{identity: fake}
	err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-one"}, nil)
	if !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("expected identity not found, got %v", err)
	}
}

func TestPatchIdentityRoleAttributesMapsPatchNotFound(t *testing.T) {
	ctx := context.Background()
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(rest_model.Attributes{"agents"}),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			return nil, &identity.PatchIdentityNotFound{}
		},
	}

	client := &Client{identity: fake}
	err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-one"}, nil)
	if !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("expected identity not found, got %v", err)
	}
}

func TestPatchIdentityRoleAttributesWrapsControllerError(t *testing.T) {
	ctx := context.Background()
	controllerErr := errors.New("controller unavailable")
	fake := &fakeIdentityService{
		detailIdentityFunc: detailIdentityWithRoleAttributes(rest_model.Attributes{"agents"}),
		patchIdentityFunc: func(params *identity.PatchIdentityParams) (*identity.PatchIdentityOK, error) {
			return nil, controllerErr
		},
	}

	client := &Client{identity: fake}
	err := client.PatchIdentityRoleAttributes(ctx, "identity-id", []string{"group-one"}, nil)
	if !errors.Is(err, controllerErr) {
		t.Fatalf("expected controller error, got %v", err)
	}
}

func detailIdentityWithRoleAttributes(roleAttributes rest_model.Attributes) func(*identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
	return func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
		return &identity.DetailIdentityOK{Payload: &rest_model.DetailIdentityEnvelope{Data: &rest_model.IdentityDetail{
			RoleAttributes: &roleAttributes,
		}}}, nil
	}
}

func assertPatchIdentityRoleAttributes(t *testing.T, params *identity.PatchIdentityParams, identityID string, expected []string) {
	t.Helper()
	if params == nil || params.ID != identityID || params.Identity == nil || params.Identity.RoleAttributes == nil {
		t.Fatalf("unexpected patch params: %#v", params)
	}
	if !reflect.DeepEqual([]string(*params.Identity.RoleAttributes), expected) {
		t.Fatalf("unexpected role attributes: %#v", *params.Identity.RoleAttributes)
	}
}

func TestGetIdentityLivenessConvertsFields(t *testing.T) {
	ctx := context.Background()
	connected := true
	fake := &fakeIdentityService{
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return &identity.DetailIdentityOK{Payload: &rest_model.DetailIdentityEnvelope{Data: &rest_model.IdentityDetail{
				Enrollment:              &rest_model.IdentityEnrollments{Ott: &rest_model.IdentityEnrollmentsOtt{JWT: "jwt"}},
				HasEdgeRouterConnection: &connected,
			}}}, nil
		},
	}

	client := &Client{identity: fake}
	liveness, err := client.GetIdentityLiveness(ctx, "identity-id")
	if err != nil {
		t.Fatalf("get identity liveness: %v", err)
	}
	if !liveness.EnrollmentPending || !liveness.HasEdgeRouterConnection {
		t.Fatalf("unexpected liveness: %#v", liveness)
	}
}

func TestListServicesConvertsFilterPaginationAndResponse(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	serviceName := "egress-rule-1"
	roles := rest_model.Attributes{"egress-services"}
	fake := &fakeServiceService{
		listServicesFunc: func(params *service.ListServicesParams) (*service.ListServicesOK, error) {
			expected := `name = "api.github.com" and name startsWith "egress-rule-" and roleAttributes contains "egress-services"`
			if params == nil || params.Filter == nil || *params.Filter != expected {
				t.Fatalf("unexpected filter: %#v", params)
			}
			if params.Limit == nil || *params.Limit != 10 || params.Offset == nil || *params.Offset != 20 {
				t.Fatalf("unexpected pagination: %#v", params)
			}
			return listServicesResponse([]*rest_model.ServiceDetail{{
				BaseEntity:     rest_model.BaseEntity{ID: &serviceID, Tags: tagsFromMap(map[string]string{"owner": "egress"})},
				Name:           &serviceName,
				RoleAttributes: &roles,
			}}, 10, 20, 31), nil
		},
	}

	client := &Client{service: fake}
	result, err := client.ListServices(ctx, ServiceListFilter{
		Name:           "api.github.com",
		NamePrefix:     "egress-rule-",
		RoleAttributes: []string{"egress-services"},
	}, 10, "20")
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if result.NextPageToken != "30" || len(result.Items) != 1 || result.Items[0].ID != serviceID || result.Items[0].Name != serviceName {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestListServicesByTagConvertsRequestAndResponse(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	serviceName := "service-name"
	roles := rest_model.Attributes{"role-one"}
	fake := &fakeServiceService{
		listServicesFunc: func(params *service.ListServicesParams) (*service.ListServicesOK, error) {
			if params == nil || params.Filter == nil || !strings.Contains(*params.Filter, "tags.owner=") {
				t.Fatalf("unexpected filter: %#v", params)
			}
			if params.Limit == nil || *params.Limit != 10 || params.Offset == nil || *params.Offset != 20 {
				t.Fatalf("unexpected pagination: %#v", params)
			}
			return listServicesResponse([]*rest_model.ServiceDetail{{
				BaseEntity:     rest_model.BaseEntity{ID: &serviceID, Tags: tagsFromMap(map[string]string{"owner": "networks"})},
				Name:           &serviceName,
				RoleAttributes: &roles,
			}}, 10, 20, 31), nil
		},
	}

	client := &Client{service: fake}
	result, err := client.ListServicesByTag(ctx, map[string]string{"owner": "networks"}, 10, "20")
	if err != nil {
		t.Fatalf("list services by tag: %v", err)
	}
	if result.NextPageToken != "30" || len(result.Items) != 1 || result.Items[0].ID != serviceID {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestListIdentitiesByTagConvertsRequestAndResponse(t *testing.T) {
	ctx := context.Background()
	identityID := "identity-id"
	identityName := "identity-name"
	roles := rest_model.Attributes{"role-one"}
	fake := &fakeIdentityService{
		listIdentitiesFunc: func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error) {
			if params == nil || params.Filter == nil || !strings.Contains(*params.Filter, "tags.owner=") {
				t.Fatalf("unexpected filter: %#v", params)
			}
			return listIdentityDetailsResponse([]*rest_model.IdentityDetail{{
				BaseEntity:     rest_model.BaseEntity{ID: &identityID, Tags: tagsFromMap(map[string]string{"owner": "networks"})},
				Name:           &identityName,
				RoleAttributes: &roles,
			}}, 50, 0, 1), nil
		},
	}

	client := &Client{identity: fake}
	result, err := client.ListIdentitiesByTag(ctx, map[string]string{"owner": "networks"}, 0, "")
	if err != nil {
		t.Fatalf("list identities by tag: %v", err)
	}
	if result.NextPageToken != "" || len(result.Items) != 1 || result.Items[0].ID != identityID {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestListServicePoliciesByTagConvertsRequestAndResponse(t *testing.T) {
	ctx := context.Background()
	policyID := "policy-id"
	policyName := "policy-name"
	policyType := rest_model.DialBindDial
	fake := &fakeServicePolicyService{
		listServicePoliciesFunc: func(params *service_policy.ListServicePoliciesParams) (*service_policy.ListServicePoliciesOK, error) {
			if params == nil || params.Filter == nil || !strings.Contains(*params.Filter, "tags.owner=") {
				t.Fatalf("unexpected filter: %#v", params)
			}
			return listServicePoliciesResponse([]*rest_model.ServicePolicyDetail{{
				BaseEntity:    rest_model.BaseEntity{ID: &policyID, Tags: tagsFromMap(map[string]string{"owner": "networks"})},
				Name:          &policyName,
				Type:          &policyType,
				IdentityRoles: rest_model.Roles{"#identity"},
				ServiceRoles:  rest_model.Roles{"#service"},
			}}, 10, 0, 1), nil
		},
	}

	client := &Client{servicePolicy: fake}
	result, err := client.ListServicePoliciesByTag(ctx, map[string]string{"owner": "networks"}, 10, "")
	if err != nil {
		t.Fatalf("list service policies by tag: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != policyID || result.Items[0].Type != "Dial" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestListServicePoliciesConvertsFilterPaginationAndResponse(t *testing.T) {
	ctx := context.Background()
	policyID := "policy-id"
	policyName := "egress-rule-agent"
	policyType := rest_model.DialBindDial
	fake := &fakeServicePolicyService{
		listServicePoliciesFunc: func(params *service_policy.ListServicePoliciesParams) (*service_policy.ListServicePoliciesOK, error) {
			expected := `name = "egress-rule-agent" and name startsWith "egress-rule-" and type = "Dial" and identityRoles contains "#agent-1" and serviceRoles contains "-rule-1"`
			if params == nil || params.Filter == nil || *params.Filter != expected {
				t.Fatalf("unexpected filter: %#v", params)
			}
			if params.Limit == nil || *params.Limit != 5 || params.Offset == nil || *params.Offset != 10 {
				t.Fatalf("unexpected pagination: %#v", params)
			}
			return listServicePoliciesResponse([]*rest_model.ServicePolicyDetail{{
				BaseEntity:    rest_model.BaseEntity{ID: &policyID, Tags: tagsFromMap(map[string]string{"owner": "egress"})},
				Name:          &policyName,
				Type:          &policyType,
				IdentityRoles: rest_model.Roles{"#agent-1"},
				ServiceRoles:  rest_model.Roles{"-rule-1"},
			}}, 5, 10, 20), nil
		},
	}

	client := &Client{servicePolicy: fake}
	result, err := client.ListServicePolicies(ctx, ServicePolicyListFilter{
		Name:          "egress-rule-agent",
		NamePrefix:    "egress-rule-",
		Type:          "Dial",
		IdentityRoles: []string{"#agent-1"},
		ServiceRoles:  []string{"-rule-1"},
	}, 5, "10")
	if err != nil {
		t.Fatalf("list service policies: %v", err)
	}
	if result.NextPageToken != "15" || len(result.Items) != 1 || result.Items[0].ID != policyID || result.Items[0].Type != "Dial" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestUpdateServicePatchesConfigsAndTags(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	serviceName := "service-name"
	roles := rest_model.Attributes{"role-one"}
	call := 0
	fakeService := &fakeServiceService{
		detailServiceFunc: func(params *service.DetailServiceParams) (*service.DetailServiceOK, error) {
			call++
			return &service.DetailServiceOK{Payload: &rest_model.DetailServiceEnvelope{Data: &rest_model.ServiceDetail{
				BaseEntity:     rest_model.BaseEntity{ID: &serviceID, Tags: tagsFromMap(map[string]string{"owner": "networks"})},
				Name:           &serviceName,
				RoleAttributes: &roles,
				Configs:        []string{"existing-config"},
			}}}, nil
		},
		listConfigFunc: func(params *service.ListServiceConfigParams) (*service.ListServiceConfigOK, error) {
			return listConfigsResponse(nil, 100, 0, 0), nil
		},
		patchServiceFunc: func(params *service.PatchServiceParams) (*service.PatchServiceOK, error) {
			if params == nil || params.Service == nil {
				t.Fatalf("expected patch service")
			}
			if !reflect.DeepEqual(params.Service.Configs, []string{"existing-config", "host-config"}) {
				t.Fatalf("unexpected configs: %#v", params.Service.Configs)
			}
			assertTags(t, params.Service.Tags, map[string]string{"owner": "networks"})
			return &service.PatchServiceOK{}, nil
		},
	}
	fakeConfig := &fakeConfigService{
		createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
			if params == nil || params.Config == nil {
				t.Fatalf("expected config create")
			}
			data, ok := params.Config.Data.(map[string]any)
			if !ok {
				t.Fatalf("expected config data map, got %#v", params.Config.Data)
			}
			assertHostV1ConfigIncludesConcreteAddressPort(t, data, "127.0.0.1", 8080)
			assertTags(t, params.Config.Tags, map[string]string{"owner": "networks"})
			return createConfigResponse("host-config"), nil
		},
	}

	client := &Client{service: fakeService, config: fakeConfig}
	updated, err := client.UpdateService(ctx, serviceID, &HostV1ConfigData{Protocol: "tcp", Address: "127.0.0.1", Port: 8080}, nil, map[string]string{"owner": "networks"}, true)
	if err != nil {
		t.Fatalf("update service: %v", err)
	}
	if call != 2 || updated.ID != serviceID {
		t.Fatalf("unexpected update result: call=%d updated=%#v", call, updated)
	}
}

func TestUpdateServiceForwardingHostConfigOmitsUnsetAddressAndPort(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	serviceName := "service-name"
	roles := rest_model.Attributes{"role-one"}
	existingConfigID := "host-config"
	hostV1TypeID := hostV1ConfigTypeID
	fakeService := &fakeServiceService{
		detailServiceFunc: func(params *service.DetailServiceParams) (*service.DetailServiceOK, error) {
			return &service.DetailServiceOK{Payload: &rest_model.DetailServiceEnvelope{Data: &rest_model.ServiceDetail{
				BaseEntity:     rest_model.BaseEntity{ID: &serviceID},
				Name:           &serviceName,
				RoleAttributes: &roles,
				Configs:        []string{existingConfigID},
			}}}, nil
		},
		listConfigFunc: func(params *service.ListServiceConfigParams) (*service.ListServiceConfigOK, error) {
			return listConfigsResponse([]*rest_model.ConfigDetail{{
				BaseEntity:   rest_model.BaseEntity{ID: &existingConfigID},
				ConfigTypeID: &hostV1TypeID,
			}}, 100, 0, 1), nil
		},
		patchServiceFunc: func(params *service.PatchServiceParams) (*service.PatchServiceOK, error) {
			if params == nil || params.Service == nil {
				t.Fatalf("expected patch service")
			}
			if !reflect.DeepEqual(params.Service.Configs, []string{existingConfigID}) {
				t.Fatalf("unexpected configs: %#v", params.Service.Configs)
			}
			return &service.PatchServiceOK{}, nil
		},
	}
	fakeConfig := &fakeConfigService{
		patchConfigFunc: func(params *config.PatchConfigParams) (*config.PatchConfigOK, error) {
			if params == nil || params.Config == nil {
				t.Fatalf("expected config patch")
			}
			data, ok := params.Config.Data.(map[string]any)
			if !ok {
				t.Fatalf("expected config data map, got %#v", params.Config.Data)
			}
			assertHostV1ConfigData(t, data, map[string]any{
				"protocol":         "tcp",
				"forwardProtocol":  true,
				"forwardAddress":   true,
				"forwardPort":      true,
				"allowedProtocols": []string{"tcp"},
				"allowedAddresses": []string{"0.0.0.0/0"},
				"allowedPortRanges": []map[string]any{{
					"low":  int32(1),
					"high": int32(65535),
				}},
			})
			return &config.PatchConfigOK{}, nil
		},
	}
	host := &HostV1ConfigData{
		Protocol:          "tcp",
		ForwardProtocol:   true,
		ForwardAddress:    true,
		ForwardPort:       true,
		AllowedProtocols:  []string{"tcp"},
		AllowedAddresses:  []string{"0.0.0.0/0"},
		AllowedPortRanges: []PortRangeData{{Low: 1, High: 65535}},
	}

	client := &Client{service: fakeService, config: fakeConfig}
	if _, err := client.UpdateService(ctx, serviceID, host, nil, nil, false); err != nil {
		t.Fatalf("update service: %v", err)
	}
}

func TestUpdateServiceTagOnlyDoesNotPatchConfigs(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	serviceName := "service-name"
	roles := rest_model.Attributes{"role-one"}
	fakeService := &fakeServiceService{
		detailServiceFunc: func(params *service.DetailServiceParams) (*service.DetailServiceOK, error) {
			return &service.DetailServiceOK{Payload: &rest_model.DetailServiceEnvelope{Data: &rest_model.ServiceDetail{
				BaseEntity:     rest_model.BaseEntity{ID: &serviceID, Tags: tagsFromMap(map[string]string{"owner": "networks"})},
				Name:           &serviceName,
				RoleAttributes: &roles,
			}}}, nil
		},
		listConfigFunc: func(params *service.ListServiceConfigParams) (*service.ListServiceConfigOK, error) {
			t.Fatalf("list service config should not be called for tag-only update")
			return nil, nil
		},
		patchServiceFunc: func(params *service.PatchServiceParams) (*service.PatchServiceOK, error) {
			if params == nil || params.Service == nil {
				t.Fatalf("expected patch service")
			}
			if params.Service.Configs != nil {
				t.Fatalf("tag-only update must not patch configs: %#v", params.Service.Configs)
			}
			assertTags(t, params.Service.Tags, map[string]string{"owner": "networks"})
			return &service.PatchServiceOK{}, nil
		},
	}
	fakeConfig := &fakeConfigService{
		createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
			t.Fatalf("create config should not be called for tag-only update")
			return nil, nil
		},
	}

	client := &Client{service: fakeService, config: fakeConfig}
	if _, err := client.UpdateService(ctx, serviceID, nil, nil, map[string]string{"owner": "networks"}, true); err != nil {
		t.Fatalf("update service: %v", err)
	}
}

func TestCreateAgentIdentityCreateFailure(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	workloadID := uuid.New()
	createErr := errors.New("create failed")
	var detailCalled bool

	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, workloadID)
			assertCreateAgentRoleAttributes(t, params, agentID, workloadID)
			return nil, createErr
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			detailCalled = true
			return nil, errors.New("detail identity should not be called")
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateAgentIdentity(ctx, agentID, workloadID)
	if err == nil {
		t.Fatalf("expected create error")
	}
	if !errors.Is(err, createErr) {
		t.Fatalf("expected error %q, got %v", createErr, err)
	}
	if detailCalled {
		t.Fatalf("expected detail not called")
	}
}

func TestCreateServiceWithConfigs(t *testing.T) {
	ctx := context.Background()

	t.Run("no configs", func(t *testing.T) {
		serviceID := "service-id"
		fakeService := &fakeServiceService{
			createServiceFunc: func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error) {
				if params == nil || params.Service == nil {
					t.Fatalf("expected service create params")
				}
				if params.Service.Name == nil || *params.Service.Name != "my-service" {
					t.Fatalf("unexpected service name: %#v", params.Service.Name)
				}
				if !reflect.DeepEqual(params.Service.RoleAttributes, []string{"role"}) {
					t.Fatalf("unexpected role attributes: %#v", params.Service.RoleAttributes)
				}
				if len(params.Service.Configs) != 0 {
					t.Fatalf("expected no configs, got %#v", params.Service.Configs)
				}
				return createServiceResponse(serviceID), nil
			},
		}
		fakeConfig := &fakeConfigService{
			createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
				t.Fatalf("create config should not be called: %#v", params)
				return nil, nil
			},
			deleteConfigFunc: func(params *config.DeleteConfigParams) (*config.DeleteConfigOK, error) {
				t.Fatalf("delete config should not be called: %#v", params)
				return nil, nil
			},
		}

		client := &Client{service: fakeService, config: fakeConfig}
		got, err := client.CreateServiceWithConfigs(ctx, "my-service", []string{"role"}, nil, nil)
		if err != nil {
			t.Fatalf("create service with configs: %v", err)
		}
		if got != serviceID {
			t.Fatalf("expected service id %q, got %q", serviceID, got)
		}
	})

	t.Run("creates configs", func(t *testing.T) {
		host := &HostV1ConfigData{
			Protocol:          "tcp",
			Address:           "127.0.0.1",
			Port:              8080,
			ForwardProtocol:   true,
			ForwardAddress:    true,
			ForwardPort:       true,
			AllowedProtocols:  []string{"tcp"},
			AllowedAddresses:  []string{"example.com"},
			AllowedPortRanges: []PortRangeData{{Low: 80, High: 443}},
		}
		intercept := &InterceptV1ConfigData{
			Protocols: []string{"tcp"},
			Addresses: []string{"example.com"},
			PortRanges: []PortRangeData{{
				Low:  80,
				High: 80,
			}},
		}
		configIDs := []string{"host-config", "intercept-config"}
		serviceID := "service-id"
		callIndex := 0

		fakeConfig := &fakeConfigService{
			createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
				if params == nil || params.Config == nil {
					t.Fatalf("expected config create params")
				}
				if params.Config.ConfigTypeID == nil || params.Config.Name == nil {
					t.Fatalf("expected config type and name")
				}
				data, ok := params.Config.Data.(map[string]any)
				if !ok {
					t.Fatalf("expected config data map, got %#v", params.Config.Data)
				}

				switch callIndex {
				case 0:
					if *params.Config.ConfigTypeID != hostV1ConfigTypeID {
						t.Fatalf("unexpected config type: %s", *params.Config.ConfigTypeID)
					}
					if *params.Config.Name != "svc-host-v1" {
						t.Fatalf("unexpected config name: %s", *params.Config.Name)
					}
					expected := map[string]any{
						"protocol":        "tcp",
						"address":         "127.0.0.1",
						"port":            int32(8080),
						"forwardProtocol": true,
						"forwardAddress":  true,
						"forwardPort":     true,
						"allowedProtocols": []string{
							"tcp",
						},
						"allowedAddresses": []string{
							"example.com",
						},
						"allowedPortRanges": []map[string]any{{
							"low":  int32(80),
							"high": int32(443),
						}},
					}
					if !reflect.DeepEqual(data, expected) {
						t.Fatalf("unexpected host config data: %#v", data)
					}
				case 1:
					if *params.Config.ConfigTypeID != interceptV1ConfigTypeID {
						t.Fatalf("unexpected config type: %s", *params.Config.ConfigTypeID)
					}
					if *params.Config.Name != "svc-intercept-v1" {
						t.Fatalf("unexpected config name: %s", *params.Config.Name)
					}
					expected := map[string]any{
						"protocols": []string{"tcp"},
						"addresses": []string{"example.com"},
						"portRanges": []map[string]any{{
							"low":  int32(80),
							"high": int32(80),
						}},
					}
					if !reflect.DeepEqual(data, expected) {
						t.Fatalf("unexpected intercept config data: %#v", data)
					}
				default:
					t.Fatalf("unexpected config create call %d", callIndex)
				}
				configID := configIDs[callIndex]
				callIndex++
				return createConfigResponse(configID), nil
			},
		}

		fakeService := &fakeServiceService{
			createServiceFunc: func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error) {
				if params == nil || params.Service == nil {
					t.Fatalf("expected service create params")
				}
				if !reflect.DeepEqual(params.Service.Configs, configIDs) {
					t.Fatalf("unexpected configs: %#v", params.Service.Configs)
				}
				return createServiceResponse(serviceID), nil
			},
		}

		client := &Client{service: fakeService, config: fakeConfig}
		got, err := client.CreateServiceWithConfigs(ctx, "svc", []string{"role"}, host, intercept)
		if err != nil {
			t.Fatalf("create service with configs: %v", err)
		}
		if got != serviceID {
			t.Fatalf("expected service id %q, got %q", serviceID, got)
		}
	})

	t.Run("forwarding host config omits unset address and port", func(t *testing.T) {
		host := &HostV1ConfigData{
			Protocol:          "tcp",
			ForwardProtocol:   true,
			ForwardAddress:    true,
			ForwardPort:       true,
			AllowedProtocols:  []string{"tcp"},
			AllowedAddresses:  []string{"0.0.0.0/0"},
			AllowedPortRanges: []PortRangeData{{Low: 1, High: 65535}},
		}
		fakeConfig := &fakeConfigService{
			createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
				if params == nil || params.Config == nil {
					t.Fatalf("expected config create params")
				}
				data, ok := params.Config.Data.(map[string]any)
				if !ok {
					t.Fatalf("expected config data map, got %#v", params.Config.Data)
				}
				assertHostV1ConfigData(t, data, map[string]any{
					"protocol":         "tcp",
					"forwardProtocol":  true,
					"forwardAddress":   true,
					"forwardPort":      true,
					"allowedProtocols": []string{"tcp"},
					"allowedAddresses": []string{"0.0.0.0/0"},
					"allowedPortRanges": []map[string]any{{
						"low":  int32(1),
						"high": int32(65535),
					}},
				})
				return createConfigResponse("host-config"), nil
			},
		}
		fakeService := &fakeServiceService{
			createServiceFunc: func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error) {
				return createServiceResponse("service-id"), nil
			},
		}

		client := &Client{service: fakeService, config: fakeConfig}
		if _, err := client.CreateServiceWithConfigs(ctx, "egress-rule", []string{"role"}, host, nil); err != nil {
			t.Fatalf("create service with configs: %v", err)
		}
	})

	t.Run("cleanup on service failure", func(t *testing.T) {
		host := &HostV1ConfigData{Protocol: "tcp", Address: "127.0.0.1", Port: 8080}
		intercept := &InterceptV1ConfigData{
			Protocols: []string{"tcp"},
			Addresses: []string{"example.com"},
			PortRanges: []PortRangeData{{
				Low:  80,
				High: 80,
			}},
		}
		serviceErr := errors.New("service create failed")
		deleted := make([]string, 0, 2)
		configIDs := []string{"host-config", "intercept-config"}
		callIndex := 0

		fakeConfig := &fakeConfigService{
			createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
				configID := configIDs[callIndex]
				callIndex++
				return createConfigResponse(configID), nil
			},
			deleteConfigFunc: func(params *config.DeleteConfigParams) (*config.DeleteConfigOK, error) {
				if params == nil {
					t.Fatalf("expected delete config params")
				}
				deleted = append(deleted, params.ID)
				return &config.DeleteConfigOK{}, nil
			},
		}

		fakeService := &fakeServiceService{
			createServiceFunc: func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error) {
				return nil, serviceErr
			},
		}

		client := &Client{service: fakeService, config: fakeConfig}
		_, err := client.CreateServiceWithConfigs(ctx, "svc", []string{"role"}, host, intercept)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !errors.Is(err, serviceErr) {
			t.Fatalf("expected service error, got %v", err)
		}
		if len(deleted) != 2 {
			t.Fatalf("expected 2 configs deleted, got %v", deleted)
		}
		deletions := map[string]bool{}
		for _, id := range deleted {
			deletions[id] = true
		}
		for _, id := range configIDs {
			if !deletions[id] {
				t.Fatalf("expected config %s deleted", id)
			}
		}
	})
}

func TestCreateServicePolicy(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		policyType string
		wantType   rest_model.DialBind
	}{
		{
			name:       "bind",
			policyType: "Bind",
			wantType:   rest_model.DialBindBind,
		},
		{
			name:       "dial",
			policyType: "Dial",
			wantType:   rest_model.DialBindDial,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeServicePolicyService{
				createServicePolicyFunc: func(params *service_policy.CreateServicePolicyParams) (*service_policy.CreateServicePolicyCreated, error) {
					if params == nil || params.Policy == nil {
						t.Fatalf("expected service policy params")
					}
					if params.Policy.Name == nil || *params.Policy.Name != "policy" {
						t.Fatalf("unexpected policy name")
					}
					if params.Policy.Type == nil || *params.Policy.Type != tc.wantType {
						t.Fatalf("unexpected policy type: %#v", params.Policy.Type)
					}
					if params.Policy.Semantic == nil || *params.Policy.Semantic != rest_model.SemanticAnyOf {
						t.Fatalf("unexpected policy semantic: %#v", params.Policy.Semantic)
					}
					if !reflect.DeepEqual(params.Policy.IdentityRoles, rest_model.Roles{"#identity"}) {
						t.Fatalf("unexpected identity roles: %#v", params.Policy.IdentityRoles)
					}
					if !reflect.DeepEqual(params.Policy.ServiceRoles, rest_model.Roles{"#service"}) {
						t.Fatalf("unexpected service roles: %#v", params.Policy.ServiceRoles)
					}
					return createServicePolicyResponse("policy-id"), nil
				},
			}

			client := &Client{servicePolicy: fake}
			policyID, err := client.CreateServicePolicy(ctx, "policy", tc.policyType, []string{"#identity"}, []string{"#service"})
			if err != nil {
				t.Fatalf("create service policy: %v", err)
			}
			if policyID != "policy-id" {
				t.Fatalf("expected policy id, got %q", policyID)
			}
		})
	}
}

func TestDeleteServicePolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		fake := &fakeServicePolicyService{
			deleteServicePolicyFunc: func(params *service_policy.DeleteServicePolicyParams) (*service_policy.DeleteServicePolicyOK, error) {
				if params == nil || params.ID != "policy-id" {
					t.Fatalf("unexpected delete params: %#v", params)
				}
				return &service_policy.DeleteServicePolicyOK{}, nil
			},
		}
		client := &Client{servicePolicy: fake}
		if err := client.DeleteServicePolicy(ctx, "policy-id"); err != nil {
			t.Fatalf("delete service policy: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		fake := &fakeServicePolicyService{
			deleteServicePolicyFunc: func(params *service_policy.DeleteServicePolicyParams) (*service_policy.DeleteServicePolicyOK, error) {
				return nil, &service_policy.DeleteServicePolicyNotFound{}
			},
		}
		client := &Client{servicePolicy: fake}
		err := client.DeleteServicePolicy(ctx, "missing")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !errors.Is(err, ErrServicePolicyNotFound) {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestCreateDeviceIdentity(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	createdID := "created-id"
	jwt := "jwt-token"

	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, userID)
			if params == nil || params.Identity == nil || params.Identity.Name == nil {
				t.Fatalf("expected identity name")
			}
			if *params.Identity.Name != "laptop" {
				t.Fatalf("unexpected identity name: %s", *params.Identity.Name)
			}
			if params.Identity.RoleAttributes == nil || !reflect.DeepEqual(*params.Identity.RoleAttributes, rest_model.Attributes{"devices"}) {
				t.Fatalf("unexpected role attributes: %#v", params.Identity.RoleAttributes)
			}
			if params.Identity.Enrollment == nil || !params.Identity.Enrollment.Ott {
				t.Fatalf("expected ott enrollment")
			}
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			if params == nil || params.ID != createdID {
				t.Fatalf("expected detail identity id %q, got %#v", createdID, params)
			}
			return detailIdentityResponse(jwt), nil
		},
	}

	client := &Client{identity: fake}
	zitiID, token, err := client.CreateDeviceIdentity(ctx, userID, "laptop")
	if err != nil {
		t.Fatalf("create device identity: %v", err)
	}
	if zitiID != createdID {
		t.Fatalf("expected identity id %q, got %q", createdID, zitiID)
	}
	if token != jwt {
		t.Fatalf("expected jwt %q, got %q", jwt, token)
	}
}

func TestCreateDeviceIdentityCreateFailure(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	createErr := errors.New("create failed")
	var detailCalled bool

	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, userID)
			return nil, createErr
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			detailCalled = true
			return nil, errors.New("detail identity should not be called")
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateDeviceIdentity(ctx, userID, "laptop")
	if err == nil {
		t.Fatalf("expected create error")
	}
	if !errors.Is(err, createErr) {
		t.Fatalf("expected error %q, got %v", createErr, err)
	}
	if detailCalled {
		t.Fatalf("expected detail not called")
	}
}

func assertCreateExternalID(t *testing.T, params *identity.CreateIdentityParams, expectedID uuid.UUID) {
	t.Helper()
	if params == nil || params.Identity == nil || params.Identity.ExternalID == nil {
		t.Fatalf("expected create identity external id")
	}
	if *params.Identity.ExternalID != expectedID.String() {
		t.Fatalf("expected external id %q, got %q", expectedID.String(), *params.Identity.ExternalID)
	}
}

func assertCreateAgentRoleAttributes(t *testing.T, params *identity.CreateIdentityParams, agentID, workloadID uuid.UUID, additional ...string) {
	t.Helper()
	if params == nil || params.Identity == nil || params.Identity.RoleAttributes == nil {
		t.Fatalf("expected create identity role attributes")
	}
	expectedRoleAttributes := rest_model.Attributes{
		roleAttributeAgents,
		"agent-" + agentID.String(),
		"workload-" + workloadID.String(),
	}
	expectedRoleAttributes = append(expectedRoleAttributes, additional...)
	if !reflect.DeepEqual(*params.Identity.RoleAttributes, expectedRoleAttributes) {
		t.Fatalf("unexpected role attributes: %#v", params.Identity.RoleAttributes)
	}
}

func assertHostV1ConfigData(t *testing.T, actual map[string]any, expected map[string]any) {
	t.Helper()
	if _, ok := actual["address"]; ok {
		t.Fatalf("host.v1 config must not include unset address: %#v", actual)
	}
	if _, ok := actual["port"]; ok {
		t.Fatalf("host.v1 config must not include unset port: %#v", actual)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected host config data: %#v", actual)
	}
}

func assertHostV1ConfigIncludesConcreteAddressPort(t *testing.T, actual map[string]any, address string, port int32) {
	t.Helper()
	if got, ok := actual["address"]; !ok || got != address {
		t.Fatalf("expected address %q, got %#v in %#v", address, got, actual)
	}
	if got, ok := actual["port"]; !ok || got != port {
		t.Fatalf("expected port %d, got %#v in %#v", port, got, actual)
	}
}

func assertTags(t *testing.T, tags *rest_model.Tags, expected map[string]string) {
	t.Helper()
	if tags == nil {
		t.Fatalf("expected tags")
	}
	if !reflect.DeepEqual(mapFromTags(tags), expected) {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func assertSameStrings(t *testing.T, actual, expected []string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
	actualSet := map[string]bool{}
	for _, value := range actual {
		actualSet[value] = true
	}
	for _, value := range expected {
		if !actualSet[value] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}

func createIdentityResponse(identityID string) *identity.CreateIdentityCreated {
	return &identity.CreateIdentityCreated{Payload: &rest_model.CreateEnvelope{Data: &rest_model.CreateLocation{ID: identityID}}}
}

func detailIdentityResponse(jwt string) *identity.DetailIdentityOK {
	expiresAt := time.Now().Add(time.Hour).UTC()
	return detailIdentityResponseWithExpiry(jwt, expiresAt)
}

func detailIdentityResponseWithExpiry(jwt string, expiresAt time.Time) *identity.DetailIdentityOK {
	expires := strfmt.DateTime(expiresAt)
	return &identity.DetailIdentityOK{Payload: &rest_model.DetailIdentityEnvelope{Data: &rest_model.IdentityDetail{Enrollment: &rest_model.IdentityEnrollments{
		Ott: &rest_model.IdentityEnrollmentsOtt{JWT: jwt, ExpiresAt: expires},
	}}}}
}

func createServiceResponse(serviceID string) *service.CreateServiceCreated {
	return &service.CreateServiceCreated{Payload: &rest_model.CreateEnvelope{Data: &rest_model.CreateLocation{ID: serviceID}}}
}

func createConfigResponse(configID string) *config.CreateConfigCreated {
	return &config.CreateConfigCreated{Payload: &rest_model.CreateEnvelope{Data: &rest_model.CreateLocation{ID: configID}}}
}

func createServicePolicyResponse(policyID string) *service_policy.CreateServicePolicyCreated {
	return &service_policy.CreateServicePolicyCreated{Payload: &rest_model.CreateEnvelope{Data: &rest_model.CreateLocation{ID: policyID}}}
}

func TestDeleteIdentityByExternalIDDeletesMatches(t *testing.T) {
	ctx := context.Background()
	externalID := uuid.New().String()
	deleted := make([]string, 0)

	fake := &fakeIdentityService{
		listIdentitiesFunc: func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error) {
			if params == nil || params.Filter == nil {
				t.Fatalf("expected filter param")
			}
			expectedFilter := "externalId=" + strconv.Quote(externalID)
			if *params.Filter != expectedFilter {
				t.Fatalf("expected filter %q, got %q", expectedFilter, *params.Filter)
			}
			return listIdentitiesResponse([]string{"id-1", "id-2"}, 100, 0, 2), nil
		},
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleted = append(deleted, params.ID)
			return &identity.DeleteIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	if err := client.deleteIdentityByExternalID(ctx, externalID); err != nil {
		t.Fatalf("delete identity by external id: %v", err)
	}
	if len(deleted) != 2 {
		t.Fatalf("expected 2 deletions, got %v", deleted)
	}
	if deleted[0] != "id-1" || deleted[1] != "id-2" {
		t.Fatalf("unexpected deletions: %v", deleted)
	}
}

func TestDeleteIdentityByExternalIDNoMatches(t *testing.T) {
	ctx := context.Background()
	externalID := uuid.New().String()
	deleteCalled := false

	fake := &fakeIdentityService{
		listIdentitiesFunc: func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error) {
			return listIdentitiesResponse(nil, 100, 0, 0), nil
		},
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleteCalled = true
			return &identity.DeleteIdentityOK{}, nil
		},
	}

	client := &Client{identity: fake}
	if err := client.deleteIdentityByExternalID(ctx, externalID); err != nil {
		t.Fatalf("delete identity by external id: %v", err)
	}
	if deleteCalled {
		t.Fatalf("expected delete not called")
	}
}

func listIdentitiesResponse(identityIDs []string, limit, offset, total int64) *identity.ListIdentitiesOK {
	data := make(rest_model.IdentityList, len(identityIDs))
	for i, identityID := range identityIDs {
		id := identityID
		data[i] = &rest_model.IdentityDetail{BaseEntity: rest_model.BaseEntity{ID: &id}}
	}
	return &identity.ListIdentitiesOK{
		Payload: &rest_model.ListIdentitiesEnvelope{
			Data: data,
			Meta: &rest_model.Meta{
				Pagination: &rest_model.Pagination{
					Limit:      &limit,
					Offset:     &offset,
					TotalCount: &total,
				},
			},
		},
	}
}

func listIdentityDetailsResponse(details []*rest_model.IdentityDetail, limit, offset, total int64) *identity.ListIdentitiesOK {
	return &identity.ListIdentitiesOK{
		Payload: &rest_model.ListIdentitiesEnvelope{
			Data: details,
			Meta: paginationMeta(limit, offset, total),
		},
	}
}

func listServicesResponse(details []*rest_model.ServiceDetail, limit, offset, total int64) *service.ListServicesOK {
	return &service.ListServicesOK{
		Payload: &rest_model.ListServicesEnvelope{
			Data: details,
			Meta: paginationMeta(limit, offset, total),
		},
	}
}

func listServicePoliciesResponse(details []*rest_model.ServicePolicyDetail, limit, offset, total int64) *service_policy.ListServicePoliciesOK {
	return &service_policy.ListServicePoliciesOK{
		Payload: &rest_model.ListServicePoliciesEnvelope{
			Data: details,
			Meta: paginationMeta(limit, offset, total),
		},
	}
}

func listConfigsResponse(details []*rest_model.ConfigDetail, limit, offset, total int64) *service.ListServiceConfigOK {
	return &service.ListServiceConfigOK{
		Payload: &rest_model.ListConfigsEnvelope{
			Data: details,
			Meta: paginationMeta(limit, offset, total),
		},
	}
}

func paginationMeta(limit, offset, total int64) *rest_model.Meta {
	return &rest_model.Meta{Pagination: &rest_model.Pagination{Limit: &limit, Offset: &offset, TotalCount: &total}}
}
