package server

import (
	"fmt"

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

func fromProtoIdentityType(value zitimanagementv1.IdentityType) (store.IdentityType, error) {
	switch value {
	case zitimanagementv1.IdentityType_IDENTITY_TYPE_AGENT:
		return store.IdentityTypeAgent, nil
	case zitimanagementv1.IdentityType_IDENTITY_TYPE_RUNNER:
		return store.IdentityTypeRunner, nil
	case zitimanagementv1.IdentityType_IDENTITY_TYPE_CHANNEL:
		return store.IdentityTypeChannel, nil
	case zitimanagementv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED:
		return store.IdentityTypeUnspecified, fmt.Errorf("identity type unspecified")
	default:
		return store.IdentityTypeUnspecified, fmt.Errorf("unknown identity type %v", value)
	}
}

func toProtoIdentityType(value store.IdentityType) (zitimanagementv1.IdentityType, error) {
	switch value {
	case store.IdentityTypeAgent:
		return zitimanagementv1.IdentityType_IDENTITY_TYPE_AGENT, nil
	case store.IdentityTypeRunner:
		return zitimanagementv1.IdentityType_IDENTITY_TYPE_RUNNER, nil
	case store.IdentityTypeChannel:
		return zitimanagementv1.IdentityType_IDENTITY_TYPE_CHANNEL, nil
	case store.IdentityTypeUnspecified:
		return zitimanagementv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED, nil
	default:
		return zitimanagementv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED, fmt.Errorf("unknown identity type %d", value)
	}
}

func toProtoManagedIdentity(identity store.ManagedIdentity) (*zitimanagementv1.ManagedIdentity, error) {
	identityType, err := toProtoIdentityType(identity.IdentityType)
	if err != nil {
		return nil, err
	}
	return &zitimanagementv1.ManagedIdentity{
		ZitiIdentityId: identity.ZitiIdentityID,
		IdentityId:     identity.IdentityID.String(),
		IdentityType:   identityType,
		TenantId:       identity.TenantID.String(),
		CreatedAt:      timestamppb.New(identity.CreatedAt),
	}, nil
}
