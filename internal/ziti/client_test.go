package ziti

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_model"
)

type fakeIdentityService struct {
	listIdentitiesFunc func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error)
	createIdentityFunc func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error)
	deleteIdentityFunc func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error)
	detailIdentityFunc func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error)
}

func (f *fakeIdentityService) ListIdentities(params *identity.ListIdentitiesParams, _ runtime.ClientAuthInfoWriter, _ ...identity.ClientOption) (*identity.ListIdentitiesOK, error) {
	if f.listIdentitiesFunc == nil {
		return nil, errors.New("list identities not stubbed")
	}
	return f.listIdentitiesFunc(params)
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

func TestCreateAgentIdentityDeletesExistingIdentity(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	existingID := "existing-id"
	createdID := "created-id"
	jwt := "jwt-token"
	var deleteCalled bool

	fake := &fakeIdentityService{
		listIdentitiesFunc: func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error) {
			assertFilter(t, params, agentID)
			return listIdentitiesResponse(existingID), nil
		},
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleteCalled = true
			if params == nil || params.ID != existingID {
				t.Fatalf("expected delete identity id %q, got %#v", existingID, params)
			}
			return &identity.DeleteIdentityOK{}, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			if !deleteCalled {
				t.Fatalf("expected delete before create")
			}
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

func TestCreateAgentIdentityIgnoresMissingDelete(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	existingID := "missing-id"
	createdID := "created-id"
	jwt := "jwt-token"
	var deleteCalled bool

	fake := &fakeIdentityService{
		listIdentitiesFunc: func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error) {
			assertFilter(t, params, agentID)
			return listIdentitiesResponse(existingID), nil
		},
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleteCalled = true
			return nil, &identity.DeleteIdentityNotFound{}
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			if !deleteCalled {
				t.Fatalf("expected delete before create")
			}
			assertCreateExternalID(t, params, agentID)
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
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

func TestCreateAgentIdentitySkipsDeleteWhenNoExistingIdentity(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	createdID := "created-id"
	jwt := "jwt-token"
	var deleteCalls int

	fake := &fakeIdentityService{
		listIdentitiesFunc: func(params *identity.ListIdentitiesParams) (*identity.ListIdentitiesOK, error) {
			assertFilter(t, params, agentID)
			return listIdentitiesResponse(""), nil
		},
		deleteIdentityFunc: func(params *identity.DeleteIdentityParams) (*identity.DeleteIdentityOK, error) {
			deleteCalls++
			return &identity.DeleteIdentityOK{}, nil
		},
		createIdentityFunc: func(params *identity.CreateIdentityParams) (*identity.CreateIdentityCreated, error) {
			assertCreateExternalID(t, params, agentID)
			return createIdentityResponse(createdID), nil
		},
		detailIdentityFunc: func(params *identity.DetailIdentityParams) (*identity.DetailIdentityOK, error) {
			return detailIdentityResponse(jwt), nil
		},
	}

	client := &Client{identity: fake}
	_, _, err := client.CreateAgentIdentity(ctx, agentID)
	if err != nil {
		t.Fatalf("create agent identity: %v", err)
	}
	if deleteCalls != 0 {
		t.Fatalf("expected delete not called, got %d", deleteCalls)
	}
}

func assertFilter(t *testing.T, params *identity.ListIdentitiesParams, agentID uuid.UUID) {
	t.Helper()
	if params == nil || params.Filter == nil {
		t.Fatalf("expected filter param set")
	}
	expected := fmt.Sprintf("externalId = \"%s\"", agentID.String())
	if *params.Filter != expected {
		t.Fatalf("expected filter %q, got %q", expected, *params.Filter)
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

func listIdentitiesResponse(identityID string) *identity.ListIdentitiesOK {
	if identityID == "" {
		return &identity.ListIdentitiesOK{Payload: &rest_model.ListIdentitiesEnvelope{Data: rest_model.IdentityList{}}}
	}
	return &identity.ListIdentitiesOK{Payload: &rest_model.ListIdentitiesEnvelope{Data: rest_model.IdentityList{
		&rest_model.IdentityDetail{BaseEntity: rest_model.BaseEntity{ID: &identityID}},
	}}}
}

func createIdentityResponse(identityID string) *identity.CreateIdentityCreated {
	return &identity.CreateIdentityCreated{Payload: &rest_model.CreateEnvelope{Data: &rest_model.CreateLocation{ID: identityID}}}
}

func detailIdentityResponse(jwt string) *identity.DetailIdentityOK {
	return &identity.DetailIdentityOK{Payload: &rest_model.DetailIdentityEnvelope{Data: &rest_model.IdentityDetail{Enrollment: &rest_model.IdentityEnrollments{
		Ott: &rest_model.IdentityEnrollmentsOtt{JWT: jwt},
	}}}}
}
