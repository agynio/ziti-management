package ziti

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
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

func assertCreateExternalID(t *testing.T, params *identity.CreateIdentityParams, agentID uuid.UUID) {
	t.Helper()
	if params == nil || params.Identity == nil || params.Identity.ExternalID == nil {
		t.Fatalf("expected create identity external id")
	}
	if *params.Identity.ExternalID != agentID.String() {
		t.Fatalf("expected external id %q, got %q", agentID.String(), *params.Identity.ExternalID)
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
