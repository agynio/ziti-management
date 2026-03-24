package server

import (
	"fmt"

	identityv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/identity/v1"
	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/agynio/ziti-management/internal/store"
)

func parseUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.UUID{}, fmt.Errorf("value is empty")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

func fromProtoIdentityType(value identityv1.IdentityType) (store.IdentityType, error) {
	switch value {
	case identityv1.IdentityType_IDENTITY_TYPE_AGENT:
		return store.IdentityTypeAgent, nil
	case identityv1.IdentityType_IDENTITY_TYPE_RUNNER:
		return store.IdentityTypeRunner, nil
	case identityv1.IdentityType_IDENTITY_TYPE_CHANNEL:
		return store.IdentityTypeChannel, nil
	case identityv1.IdentityType_IDENTITY_TYPE_APP:
		return store.IdentityTypeApp, nil
	case identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED:
		return store.IdentityTypeUnspecified, fmt.Errorf("identity type unspecified")
	default:
		return store.IdentityTypeUnspecified, fmt.Errorf("unknown identity type %v", value)
	}
}

func fromProtoServiceType(value zitimanagementv1.ServiceType) (store.ServiceType, error) {
	switch value {
	case zitimanagementv1.ServiceType_SERVICE_TYPE_GATEWAY:
		return store.ServiceTypeGateway, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_ORCHESTRATOR:
		return store.ServiceTypeOrchestrator, nil
	case zitimanagementv1.ServiceType(store.ServiceTypeRunner):
		return store.ServiceTypeRunner, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_LLM_PROXY:
		return store.ServiceTypeLLMProxy, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_UNSPECIFIED:
		return store.ServiceTypeUnspecified, fmt.Errorf("service type unspecified")
	default:
		return store.ServiceTypeUnspecified, fmt.Errorf("unknown service type %v", value)
	}
}

func toProtoIdentityType(value store.IdentityType) (identityv1.IdentityType, error) {
	switch value {
	case store.IdentityTypeAgent:
		return identityv1.IdentityType_IDENTITY_TYPE_AGENT, nil
	case store.IdentityTypeRunner:
		return identityv1.IdentityType_IDENTITY_TYPE_RUNNER, nil
	case store.IdentityTypeChannel:
		return identityv1.IdentityType_IDENTITY_TYPE_CHANNEL, nil
	case store.IdentityTypeApp:
		return identityv1.IdentityType_IDENTITY_TYPE_APP, nil
	case store.IdentityTypeUnspecified:
		return identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED, nil
	default:
		return identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED, fmt.Errorf("unknown identity type %d", value)
	}
}

func toProtoManagedIdentity(identity store.ManagedIdentity) (*zitimanagementv1.ManagedIdentity, error) {
	identityType, err := toProtoIdentityType(identity.IdentityType)
	if err != nil {
		return nil, err
	}
	zitiServiceID := ""
	if identity.ZitiServiceID != nil {
		zitiServiceID = *identity.ZitiServiceID
	}
	protoIdentity := &zitimanagementv1.ManagedIdentity{
		ZitiIdentityId: identity.ZitiIdentityID,
		IdentityId:     identity.IdentityID.String(),
		IdentityType:   identityType,
		ZitiServiceId:  zitiServiceID,
		CreatedAt:      timestamppb.New(identity.CreatedAt),
	}
	return protoIdentity, nil
}
