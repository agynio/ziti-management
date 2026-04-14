package ziti

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_model"
)

type fakeIdentityService struct {
	createIdentityFunc func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error)
	deleteIdentityFunc func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error)
	detailIdentityFunc func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error)
	listIdentitiesFunc func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error)
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

type fakeServiceService struct {
	createServiceFunc func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error)
	deleteServiceFunc func(params *service.DeleteServiceParams) (*service.DeleteServiceOK, error)
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

type fakeConfigService struct {
	createConfigFunc     func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error)
	createConfigTypeFunc func(params *config.CreateConfigTypeParams) (*config.CreateConfigTypeCreated, error)
	deleteConfigFunc     func(params *config.DeleteConfigParams) (*config.DeleteConfigOK, error)
	listConfigTypesFunc  func(params *config.ListConfigTypesParams) (*config.ListConfigTypesOK, error)
}

func (f *fakeConfigService) CreateConfig(params *config.CreateConfigParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.CreateConfigCreated, error) {
	if f.createConfigFunc == nil {
		return nil, errors.New("create config not stubbed")
	}
	return f.createConfigFunc(params)
}

func (f *fakeConfigService) CreateConfigType(params *config.CreateConfigTypeParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.CreateConfigTypeCreated, error) {
	if f.createConfigTypeFunc == nil {
		return nil, errors.New("create config type not stubbed")
	}
	return f.createConfigTypeFunc(params)
}

func (f *fakeConfigService) DeleteConfig(params *config.DeleteConfigParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.DeleteConfigOK, error) {
	if f.deleteConfigFunc == nil {
		return nil, errors.New("delete config not stubbed")
	}
	return f.deleteConfigFunc(params)
}

func (f *fakeConfigService) ListConfigTypes(params *config.ListConfigTypesParams, _ runtime.ClientAuthInfoWriter, _ ...config.ClientOption) (*config.ListConfigTypesOK, error) {
	if f.listConfigTypesFunc == nil {
		return nil, errors.New("list config types not stubbed")
	}
	return f.listConfigTypesFunc(params)
}

type fakeServicePolicyService struct {
	createServicePolicyFunc func(params *service_policy.CreateServicePolicyParams) (*service_policy.CreateServicePolicyCreated, error)
	deleteServicePolicyFunc func(params *service_policy.DeleteServicePolicyParams) (*service_policy.DeleteServicePolicyOK, error)
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
		host := &HostV1ConfigData{Protocol: "tcp", Address: "127.0.0.1", Port: 8080}
		intercept := &InterceptV1ConfigData{
			Protocols: []string{"tcp"},
			Addresses: []string{"example.com"},
			PortRanges: []PortRangeData{{
				Low:  80,
				High: 80,
			}},
		}
		configIDs := []string{"host-config", "intercept-config"}
		hostTypeID := hostV1ConfigTypeID
		interceptTypeID := interceptV1ConfigTypeID
		hostTypeName := hostV1ConfigType
		interceptTypeName := interceptV1ConfigType
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
					if *params.Config.ConfigTypeID != hostTypeID {
						t.Fatalf("unexpected config type: %s", *params.Config.ConfigTypeID)
					}
					if *params.Config.Name != "svc-host-v1" {
						t.Fatalf("unexpected config name: %s", *params.Config.Name)
					}
					expected := map[string]any{
						"protocol": "tcp",
						"address":  "127.0.0.1",
						"port":     int32(8080),
					}
					if !reflect.DeepEqual(data, expected) {
						t.Fatalf("unexpected host config data: %#v", data)
					}
				case 1:
					if *params.Config.ConfigTypeID != interceptTypeID {
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
			createConfigTypeFunc: func(params *config.CreateConfigTypeParams) (*config.CreateConfigTypeCreated, error) {
				t.Fatalf("create config type should not be called: %#v", params)
				return nil, nil
			},
			listConfigTypesFunc: func(params *config.ListConfigTypesParams) (*config.ListConfigTypesOK, error) {
				return &config.ListConfigTypesOK{Payload: &rest_model.ListConfigTypesEnvelope{
					Data: rest_model.ConfigTypeList{
						{
							BaseEntity: rest_model.BaseEntity{ID: &hostTypeID},
							Name:       &hostTypeName,
						},
						{
							BaseEntity: rest_model.BaseEntity{ID: &interceptTypeID},
							Name:       &interceptTypeName,
						},
					},
					Meta: &rest_model.Meta{},
				}}, nil
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
		hostTypeID := hostV1ConfigTypeID
		interceptTypeID := interceptV1ConfigTypeID
		hostTypeName := hostV1ConfigType
		interceptTypeName := interceptV1ConfigType
		callIndex := 0

		fakeConfig := &fakeConfigService{
			createConfigFunc: func(params *config.CreateConfigParams) (*config.CreateConfigCreated, error) {
				configID := configIDs[callIndex]
				callIndex++
				return createConfigResponse(configID), nil
			},
			createConfigTypeFunc: func(params *config.CreateConfigTypeParams) (*config.CreateConfigTypeCreated, error) {
				t.Fatalf("create config type should not be called: %#v", params)
				return nil, nil
			},
			deleteConfigFunc: func(params *config.DeleteConfigParams) (*config.DeleteConfigOK, error) {
				if params == nil {
					t.Fatalf("expected delete config params")
				}
				deleted = append(deleted, params.ID)
				return &config.DeleteConfigOK{}, nil
			},
			listConfigTypesFunc: func(params *config.ListConfigTypesParams) (*config.ListConfigTypesOK, error) {
				return &config.ListConfigTypesOK{Payload: &rest_model.ListConfigTypesEnvelope{
					Data: rest_model.ConfigTypeList{
						{
							BaseEntity: rest_model.BaseEntity{ID: &hostTypeID},
							Name:       &hostTypeName,
						},
						{
							BaseEntity: rest_model.BaseEntity{ID: &interceptTypeID},
							Name:       &interceptTypeName,
						},
					},
					Meta: &rest_model.Meta{},
				}}, nil
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

func assertCreateAgentRoleAttributes(t *testing.T, params *identity.CreateIdentityParams, agentID, workloadID uuid.UUID) {
	t.Helper()
	if params == nil || params.Identity == nil || params.Identity.RoleAttributes == nil {
		t.Fatalf("expected create identity role attributes")
	}
	expectedRoleAttributes := rest_model.Attributes{
		roleAttributeAgents,
		"agent-" + agentID.String(),
		"workload-" + workloadID.String(),
	}
	if !reflect.DeepEqual(*params.Identity.RoleAttributes, expectedRoleAttributes) {
		t.Fatalf("unexpected role attributes: %#v", params.Identity.RoleAttributes)
	}
}

func createIdentityResponse(identityID string) *identity.CreateIdentityCreated {
	return &identity.CreateIdentityCreated{Payload: &rest_model.CreateEnvelope{Data: &rest_model.CreateLocation{ID: identityID}}}
}

func detailIdentityResponse(jwt string) *identity.DetailIdentityOK {
	return &identity.DetailIdentityOK{Payload: &rest_model.DetailIdentityEnvelope{Data: &rest_model.IdentityDetail{Enrollment: &rest_model.IdentityEnrollments{
		Ott: &rest_model.IdentityEnrollmentsOtt{JWT: jwt},
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
