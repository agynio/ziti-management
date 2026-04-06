package ziti

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_model"
	sdkziti "github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
)

type fakeIdentityService struct {
	createIdentityFunc func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error)
	deleteIdentityFunc func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error)
	detailIdentityFunc func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error)
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

func TestCreateAgentIdentityCreatesIdentity(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	createdID := "created-id"
	jwt := "jwt-token"

	fake := &fakeIdentityService{
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			t.Fatalf("delete identity should not be called: %#v", params)
			return nil, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, agentID)
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
	zitiID, token, err := client.CreateAgentIdentity(ctx, agentID)
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
	createErr := errors.New("create failed")
	var detailCalled bool

	fake := &fakeIdentityService{
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, agentID)
			return nil, createErr
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			detailCalled = true
			return nil, errors.New("detail identity should not be called")
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateAgentIdentity(ctx, agentID)
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

func TestCreateAndEnrollRunnerIdentity(t *testing.T) {
	ctx := context.Background()
	runnerID := uuid.New()
	createdID := "runner-created-id"
	jwtToken := "jwt-token"
	roleAttributes := []string{"runner-services", "extra"}
	enrollCalled := false

	stubClientEnrollment(t, func(token string) (*sdkziti.EnrollmentClaims, *jwt.Token, error) {
		if token != jwtToken {
			t.Fatalf("expected jwt %q, got %q", jwtToken, token)
		}
		return &sdkziti.EnrollmentClaims{}, nil, nil
	}, func(flags enroll.EnrollmentFlags) (*sdkziti.Config, error) {
		enrollCalled = true
		if flags.Token == nil {
			t.Fatalf("expected enrollment claims")
		}
		if !flags.KeyAlg.EC() {
			t.Fatalf("expected EC key algorithm")
		}
		return &sdkziti.Config{}, nil
	})

	fake := &fakeIdentityService{
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			t.Fatalf("delete identity should not be called: %#v", params)
			return nil, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateIdentityParams(t, params, runnerID, roleAttributes, "runner-"+runnerID.String()+"-")
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			if params == nil || params.ID != createdID {
				t.Fatalf("expected detail identity id %q, got %#v", createdID, params)
			}
			return detailIdentityResponse(jwtToken), nil
		},
	}

	client := &Client{identity: fake}
	zitiID, identityJSON, err := client.CreateAndEnrollRunnerIdentity(ctx, runnerID, roleAttributes)
	if err != nil {
		t.Fatalf("create runner identity: %v", err)
	}
	if zitiID != createdID {
		t.Fatalf("expected identity id %q, got %q", createdID, zitiID)
	}
	if len(identityJSON) == 0 {
		t.Fatalf("expected identity json")
	}
	if !enrollCalled {
		t.Fatalf("expected enroll called")
	}
}

func TestCreateAndEnrollRunnerIdentityEnrollFailureCleansUp(t *testing.T) {
	ctx := context.Background()
	runnerID := uuid.New()
	createdID := "runner-created-id"
	jwtToken := "jwt-token"
	roleAttributes := []string{"runner-services"}
	parseErr := errors.New("parse failed")
	deleteCalled := false

	stubClientEnrollment(t, func(token string) (*sdkziti.EnrollmentClaims, *jwt.Token, error) {
		if token != jwtToken {
			t.Fatalf("expected jwt %q, got %q", jwtToken, token)
		}
		return nil, nil, parseErr
	}, func(flags enroll.EnrollmentFlags) (*sdkziti.Config, error) {
		t.Fatalf("enroll should not be called")
		return nil, nil
	})

	fake := &fakeIdentityService{
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleteCalled = true
			if params == nil || params.ID != createdID {
				t.Fatalf("expected delete identity id %q, got %#v", createdID, params)
			}
			return &identity.DeleteIdentityOK{}, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateIdentityParams(t, params, runnerID, roleAttributes, "runner-"+runnerID.String()+"-")
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return detailIdentityResponse(jwtToken), nil
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateAndEnrollRunnerIdentity(ctx, runnerID, roleAttributes)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "parse enrollment token") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected cleanup delete")
	}
}

func TestCreateAndEnrollAppIdentity(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	createdID := "app-created-id"
	slug := "example"
	jwtToken := "jwt-token"
	enrollCalled := false

	stubClientEnrollment(t, func(token string) (*sdkziti.EnrollmentClaims, *jwt.Token, error) {
		if token != jwtToken {
			t.Fatalf("expected jwt %q, got %q", jwtToken, token)
		}
		return &sdkziti.EnrollmentClaims{}, nil, nil
	}, func(flags enroll.EnrollmentFlags) (*sdkziti.Config, error) {
		enrollCalled = true
		if flags.Token == nil {
			t.Fatalf("expected enrollment claims")
		}
		if !flags.KeyAlg.EC() {
			t.Fatalf("expected EC key algorithm")
		}
		return &sdkziti.Config{}, nil
	})

	fake := &fakeIdentityService{
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			t.Fatalf("delete identity should not be called: %#v", params)
			return nil, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateIdentityParams(t, params, appID, []string{"apps"}, "app-"+slug+"-")
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			if params == nil || params.ID != createdID {
				t.Fatalf("expected detail identity id %q, got %#v", createdID, params)
			}
			return detailIdentityResponse(jwtToken), nil
		},
	}

	client := &Client{identity: fake}
	zitiID, identityJSON, err := client.CreateAndEnrollAppIdentity(ctx, appID, slug)
	if err != nil {
		t.Fatalf("create app identity: %v", err)
	}
	if zitiID != createdID {
		t.Fatalf("expected identity id %q, got %q", createdID, zitiID)
	}
	if len(identityJSON) == 0 {
		t.Fatalf("expected identity json")
	}
	if !enrollCalled {
		t.Fatalf("expected enroll called")
	}
}

func TestCreateAndEnrollAppIdentityEnrollFailureCleansUp(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	createdID := "app-created-id"
	slug := "example"
	jwtToken := "jwt-token"
	parseErr := errors.New("parse failed")
	deleteCalled := false

	stubClientEnrollment(t, func(token string) (*sdkziti.EnrollmentClaims, *jwt.Token, error) {
		if token != jwtToken {
			t.Fatalf("expected jwt %q, got %q", jwtToken, token)
		}
		return nil, nil, parseErr
	}, func(flags enroll.EnrollmentFlags) (*sdkziti.Config, error) {
		t.Fatalf("enroll should not be called")
		return nil, nil
	})

	fake := &fakeIdentityService{
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleteCalled = true
			if params == nil || params.ID != createdID {
				t.Fatalf("expected delete identity id %q, got %#v", createdID, params)
			}
			return &identity.DeleteIdentityOK{}, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateIdentityParams(t, params, appID, []string{"apps"}, "app-"+slug+"-")
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return detailIdentityResponse(jwtToken), nil
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateAndEnrollAppIdentity(ctx, appID, slug)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "parse enrollment token") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected cleanup delete")
	}
}

func TestCreateService(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	name := "runner-abc"
	roles := []string{"runner-services"}

	fake := &fakeServiceService{
		createServiceFunc: func(params *service.CreateServiceParams) (*service.CreateServiceCreated, error) {
			if params == nil || params.Service == nil || params.Service.Name == nil {
				t.Fatalf("expected service name")
			}
			if *params.Service.Name != name {
				t.Fatalf("expected name %q, got %q", name, *params.Service.Name)
			}
			if !reflect.DeepEqual(params.Service.RoleAttributes, roles) {
				t.Fatalf("expected roles %v, got %v", roles, params.Service.RoleAttributes)
			}
			if params.Service.EncryptionRequired == nil || !*params.Service.EncryptionRequired {
				t.Fatalf("expected encryption required")
			}
			return createServiceResponse(serviceID), nil
		},
	}

	client := &Client{service: fake}
	createdID, err := client.CreateService(ctx, name, roles)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	if createdID != serviceID {
		t.Fatalf("expected service id %q, got %q", serviceID, createdID)
	}
}

func TestDeleteService(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"
	deleteCalled := false

	fake := &fakeServiceService{
		deleteServiceFunc: func(params *service.DeleteServiceParams) (*service.DeleteServiceOK, error) {
			deleteCalled = true
			if params == nil || params.ID != serviceID {
				t.Fatalf("expected delete service id %q, got %#v", serviceID, params)
			}
			return &service.DeleteServiceOK{}, nil
		},
	}

	client := &Client{service: fake}
	if err := client.DeleteService(ctx, serviceID); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected delete called")
	}
}

func TestDeleteServiceNotFound(t *testing.T) {
	ctx := context.Background()
	serviceID := "service-id"

	fake := &fakeServiceService{
		deleteServiceFunc: func(params *service.DeleteServiceParams) (*service.DeleteServiceOK, error) {
			return nil, &service.DeleteServiceNotFound{}
		},
	}

	client := &Client{service: fake}
	if err := client.DeleteService(ctx, serviceID); err == nil {
		t.Fatalf("expected error")
	} else if !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

type unauthorizedError struct{}

func (unauthorizedError) Error() string {
	return "unauthorized"
}

func (unauthorizedError) IsCode(code int) bool {
	return code == 401
}

func TestWithReauthRetriesOnUnauthorized(t *testing.T) {
	client := &Client{}
	reauthCalled := 0
	client.reauthenticateFn = func() error {
		reauthCalled++
		return nil
	}

	callCount := 0
	err := client.withReauth(func() error {
		callCount++
		if callCount == 1 {
			return unauthorizedError{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withReauth returned error: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 attempts, got %d", callCount)
	}
	if reauthCalled != 1 {
		t.Fatalf("expected reauth once, got %d", reauthCalled)
	}
}

func assertCreateExternalID(t *testing.T, params *identity.CreateIdentityParams, agentID uuid.UUID) {
	t.Helper()
	if params == nil || params.Identity == nil || params.Identity.ExternalID == nil {
		t.Fatalf("expected create identity external id")
	}
	if *params.Identity.ExternalID != agentID.String() {
		t.Fatalf("expected external id %q, got %q", agentID.String(), *params.Identity.ExternalID)
	}
}

func assertCreateIdentityParams(t *testing.T, params *identity.CreateIdentityParams, identityID uuid.UUID, roles []string, namePrefix string) {
	t.Helper()
	if params == nil || params.Identity == nil || params.Identity.Name == nil {
		t.Fatalf("expected identity params")
	}
	if !strings.HasPrefix(*params.Identity.Name, namePrefix) {
		t.Fatalf("expected name prefix %q, got %q", namePrefix, *params.Identity.Name)
	}
	if params.Identity.ExternalID == nil {
		t.Fatalf("expected external id")
	}
	if *params.Identity.ExternalID != identityID.String() {
		t.Fatalf("expected external id %q, got %q", identityID.String(), *params.Identity.ExternalID)
	}
	if params.Identity.RoleAttributes == nil {
		t.Fatalf("expected role attributes")
	}
	if !reflect.DeepEqual([]string(*params.Identity.RoleAttributes), roles) {
		t.Fatalf("expected roles %v, got %v", roles, []string(*params.Identity.RoleAttributes))
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

func stubClientEnrollment(
	t *testing.T,
	parse func(string) (*sdkziti.EnrollmentClaims, *jwt.Token, error),
	enrollFn func(enroll.EnrollmentFlags) (*sdkziti.Config, error),
) {
	t.Helper()

	originalParse := parseEnrollmentToken
	originalEnroll := enrollIdentity
	parseEnrollmentToken = parse
	enrollIdentity = enrollFn
	t.Cleanup(func() {
		parseEnrollmentToken = originalParse
		enrollIdentity = originalEnroll
	})
}
